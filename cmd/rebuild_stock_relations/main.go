package main

import (
	"context"
	"flag"
	"log"
	"time"

	"stock/config"
	"stock/dal/db"
	"stock/internal/pkg/converter"
	"stock/model/dal_model"
)

var (
	relationRepo = db.NewStockTopicRelationRepository()
	topicRepo    = db.NewTopicRepository()
)

func main() {
	startDate := flag.String("start", "2023-01-01", "起始日期 YYYY-MM-DD")
	endDate := flag.String("end", "", "结束日期 YYYY-MM-DD，默认为今天")
	dryRun := flag.Bool("dry-run", false, "仅打印统计信息，不写入数据库")
	flag.Parse()

	if *endDate == "" {
		*endDate = time.Now().Format("2006-01-02")
	}

	ctx := context.Background()
	if err := config.Init("config/config.yaml"); err != nil {
		log.Fatalf("init config: %v", err)
	}
	db.Init()

	log.Printf("aggregating jiuyan_raw_data from %s to %s ...", *startDate, *endDate)

	var rows []struct {
		TopicName string
		StockCode string
		HitCount  int
		FirstSeen time.Time
		LastSeen  time.Time
	}
	err := db.PostgresStockDB(ctx).
		Raw(`
			SELECT topic_name, stock_code,
				COUNT(*) AS hit_count,
				MIN(date) AS first_seen,
				MAX(date) AS last_seen
			FROM jiuyan_raw_data
			WHERE date >= ? AND date <= ?
			GROUP BY topic_name, stock_code
		`, *startDate, *endDate).
		Scan(&rows).Error
	if err != nil {
		log.Fatalf("query aggregated raw data: %v", err)
	}

	log.Printf("found %d unique (topic, stock) pairs", len(rows))

	topicNames := make([]string, 0, len(rows))
	seenTopics := make(map[string]struct{})
	for _, r := range rows {
		if _, ok := seenTopics[r.TopicName]; !ok {
			seenTopics[r.TopicName] = struct{}{}
			topicNames = append(topicNames, r.TopicName)
		}
	}

	topicNameToID := make(map[string]int64)
	if len(topicNames) > 0 {
		topics, err := topicRepo.GetByNames(ctx, topicNames)
		if err != nil {
			log.Fatalf("query topics: %v", err)
		}
		for _, t := range topics {
			topicNameToID[t.Name] = t.ID
		}
	}

	var mappings []dal_model.StockTopicRelation
	missingTopic := 0
	missingStock := 0

	for _, r := range rows {
		topicID, ok := topicNameToID[r.TopicName]
		if !ok {
			missingTopic++
			continue
		}
		tsCode := converter.JiuyanToTushare(r.StockCode)
		if tsCode == "" {
			missingStock++
			continue
		}
		mappings = append(mappings, dal_model.StockTopicRelation{
			TsCode:        tsCode,
			TopicID:       topicID,
			Source:        "jiuyan",
			TopicName:     r.TopicName,
			HitCount:      r.HitCount,
			LastSeenDate:  &r.LastSeen,
			FirstSeenDate: &r.FirstSeen,
		})
	}

	log.Printf("skipping %d pairs (topic not found), %d pairs (stock code invalid)", missingTopic, missingStock)

	if *dryRun {
		for _, m := range mappings {
			log.Printf("  ts_code=%s topic_id=%d hit=%d", m.TsCode, m.TopicID, m.HitCount)
		}
		return
	}

	log.Printf("upserting %d mappings ...", len(mappings))

	const batchSize = 500
	for i := 0; i < len(mappings); i += batchSize {
		end := i + batchSize
		if end > len(mappings) {
			end = len(mappings)
		}
		batch := mappings[i:end]
		if err := relationRepo.BulkUpsertAccumulate(ctx, batch); err != nil {
			log.Printf("upsert [%d:%d] failed: %v", i, end, err)
		} else {
			log.Printf("upsert [%d:%d] ok (%d rows)", i, end, len(batch))
		}
	}

	log.Println("done")
}
