package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"stock/config"
	"stock/dal/db"
)

func main() {

	dryRun := flag.Bool("dry-run", false, "仅打印统计信息，不写入数据库")
	flag.Parse()

	ctx := context.Background()
	if err := config.Init("config/config.yaml"); err != nil {
		log.Fatalf("init config: %v", err)
	}
	db.Init()

	topicDictRepo := db.NewTopicDictionaryRepository()
	topicDictMap, err := topicDictRepo.GetAllMap(ctx)
	if err != nil {
		log.Fatalf("load topic_dictionary: %v", err)
	}
	log.Printf("loaded %d topic_dictionary entries", len(topicDictMap))

	var rows []struct {
		TopicName   string
		FirstSeen   time.Time
		LastSeen    time.Time
		Occurrences int
	}
	err = db.PostgresStockDB(ctx).
		Raw(`
			SELECT topic_name,
				MIN(date) AS first_seen,
				MAX(date) AS last_seen,
				COUNT(*) AS occurrences
			FROM jiuyan_raw_data
			GROUP BY topic_name
		`).
		Scan(&rows).Error
	if err != nil {
		log.Fatalf("query jiuyan_raw_data: %v", err)
	}
	log.Printf("found %d distinct topic_names in jiuyan_raw_data", len(rows))

	type topicWrite struct {
		RawName    string
		NormName   string
		Category   string
		FirstSeen  time.Time
		LastSeen   time.Time
		Occurrence int
	}
	var toWrite []topicWrite
	unnormalized := 0

	for _, r := range rows {
		normName := r.TopicName
		category := ""
		if dict, ok := topicDictMap[r.TopicName]; ok {
			normName = dict.NormalizedName
			category = dict.Category
		} else {
			fmt.Printf("unnormalized %s\n", r.TopicName)
			unnormalized++
		}
		toWrite = append(toWrite, topicWrite{
			RawName:    r.TopicName,
			NormName:   normName,
			Category:   category,
			FirstSeen:  r.FirstSeen,
			LastSeen:   r.LastSeen,
			Occurrence: r.Occurrences,
		})
	}
	log.Printf("%d topics will be normalized, %d topics have no dictionary entry", len(toWrite)-unnormalized, unnormalized)

	if *dryRun {
		for _, t := range toWrite {
			if t.RawName != t.NormName {
				log.Printf("  %s -> %s [%s] (%d occurrences)", t.RawName, t.NormName, t.Category, t.Occurrence)
			}
		}
		return
	}

	normalizedCount := 0
	for _, t := range toWrite {
		if t.RawName != t.NormName {
			normalizedCount++
		}
	}
	log.Printf("actually writing %d topics (%d normalized)", len(toWrite), normalizedCount)

	success := 0
	failed := 0
	for _, t := range toWrite {
		result := db.PostgresStockDB(ctx).
			Exec(`
				INSERT INTO topics (name, category, source, first_seen_date, last_seen_date, occurrence_count, updated_at)
				VALUES (?, ?, 'jiuyan', ?, ?, ?, NOW())
				ON CONFLICT (name) DO UPDATE SET
					category = EXCLUDED.category,
					last_seen_date = GREATEST(topics.last_seen_date, EXCLUDED.last_seen_date),
					occurrence_count = topics.occurrence_count + EXCLUDED.occurrence_count,
					updated_at = NOW()
			`, t.NormName, t.Category, t.FirstSeen, t.LastSeen, t.Occurrence)
		if err := result.Error; err != nil {
			log.Printf("upsert topic %s failed: %v", t.NormName, err)
			failed++
		} else {
			success++
		}
	}

	log.Printf("done: %d success, %d failed", success, failed)
}
