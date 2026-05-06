package snapshot

import (
	"context"
	"fmt"
	"stock/external/tushare"
	"strings"
	"time"

	"stock/config"
	"stock/dal/cache"
	"stock/dal/dao"
	"stock/dal/repo"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"
	"stock/internal/pkg/utils"
	"stock/internal/service/classify"
	"stock/model/dal_model"

	"go.uber.org/zap"
)

type SnapshotServiceImpl struct {
}

func NewSnapshotService() *SnapshotServiceImpl {
	return &SnapshotServiceImpl{}
}

func buildSnapshotRecord(date time.Time, tsCode, stockName string, topicID *int64, changePct float64, isLimitUp bool, limitTimes int16, consecutiveDays int16, totalMv *float64, vol float64, amount float64) dal_model.DailyStockSnapshot {
	return dal_model.DailyStockSnapshot{
		Date:                  date,
		TsCode:                tsCode,
		StockName:             stockName,
		TopicID:               topicID,
		ChangePct:             &changePct,
		IsLimitUp:             isLimitUp,
		LimitTimes:            limitTimes,
		ConsecutiveStrongDays: consecutiveDays,
		TotalMv:               totalMv,
		Vol:                   &vol,
		Amount:                &amount,
	}
}

func (s *SnapshotServiceImpl) TakeSnapshot(ctx context.Context, date string) error {
	snapshotRepo := repo.NewSnapshotRepository()
	stockRepo := repo.NewStockRepository()

	// Step 1: Fetch daily quotes from tushare
	tradeDate := strings.ReplaceAll(date, "-", "")
	tushareClient := tushare.NewClient(&config.GlobalConfig.Tushare, config.GlobalConfig.Retry)

	quotes, err := tushareClient.DailyAll(ctx, tradeDate)
	if err != nil {
		return fmt.Errorf("fetch daily quotes: %w", err)
	}

	// Step 2: Keep the wider eligible universe for by-heat classification; final output still only persists >5% stocks.
	candidateCodes := make([]string, 0)
	quoteMap := make(map[string]*tushare.QuoteItem)
	allHeatInputs := make([]classify.StockQuoteInput, 0, len(quotes))
	for _, q := range quotes {
		if q.PctChg > classify.HeatRisingThreshold {
			allHeatInputs = append(allHeatInputs, classify.StockQuoteInput{
				TsCode:        q.TsCode,
				ChangePercent: q.PctChg,
			})
		}
		if q.PctChg >= 5 {
			candidateCodes = append(candidateCodes, q.TsCode)
			quoteMap[q.TsCode] = q
		}
	}

	if len(candidateCodes) == 0 {
		logger.Info("no stocks with change_pct >= 5", zap.String("date", date))
		return nil
	}

	// Step 3: Fetch limit-up data
	limitItems, err := tushareClient.LimitListD(ctx, tradeDate)
	if err != nil {
		logger.Warn("fetch limit_list_d failed, proceeding without limit data", zap.Error(err))
	}
	limitMap := make(map[string]*tushare.LimitListItem, len(limitItems))
	for i := range limitItems {
		limitMap[limitItems[i].TsCode] = &limitItems[i]
	}

	// Step 4: Fetch market value data
	basicItems, err := tushareClient.DailyBasic(ctx, tradeDate)
	if err != nil {
		logger.Warn("fetch daily_basic failed, proceeding without market value", zap.Error(err))
	}
	mvMap := make(map[string]float64, len(basicItems))
	for _, item := range basicItems {
		mvMap[item.TsCode] = item.TotalMv
	}

	// Step 5: Get stock names
	stocks, _ := stockRepo.GetByTsCodes(ctx, candidateCodes)
	stockNameMap := make(map[string]string, len(stocks))
	for _, st := range stocks {
		stockNameMap[st.TsCode] = st.Name
	}

	// Step 6: Classify stocks — prefer LLM results, fall back to heat classification.
	boardRules := map[dal_model.BoardCode]*dal_model.BoardRule{
		dal_model.BoardMain: {BoardCode: dal_model.BoardMain, LimitUpRatio: 0.10},
		dal_model.BoardGEM:  {BoardCode: dal_model.BoardGEM, LimitUpRatio: 0.20},
		dal_model.BoardSTAR: {BoardCode: dal_model.BoardSTAR, LimitUpRatio: 0.20},
		dal_model.BoardBSE:  {BoardCode: dal_model.BoardBSE, LimitUpRatio: 0.30},
	}
	classifySvc := classify.NewClassifyService()
	stockInfoMap := make(map[string]*dal_model.StockBasicInfo, len(stocks))
	for i := range stocks {
		stockInfoMap[stocks[i].TsCode] = &stocks[i]
	}
	for i := range allHeatInputs {
		input := &allHeatInputs[i]
		name := stockNameMap[input.TsCode]
		isLimitUp := false
		if _, ok := limitMap[input.TsCode]; ok {
			isLimitUp = true
		} else {
			quote := quoteMap[input.TsCode]
			if quote == nil {
				for _, q := range quotes {
					if q.TsCode == input.TsCode {
						quote = q
						break
					}
				}
			}
			if quote != nil {
				board := limiter.DetectBoard(input.TsCode[:6])
				rule := boardRules[board]
				output := limiter.DetectLimitUp(dal_model.DetectInput{
					TsCode:       input.TsCode,
					StockName:    name,
					CurrentPrice: quote.Price,
					PreClose:     quote.PreClose,
					ChangePct:    quote.PctChg,
					BoardCode:    board,
					IsST:         stockInfoMap[input.TsCode] != nil && stockInfoMap[input.TsCode].IsST,
				}, rule)
				isLimitUp = output.IsLimitUp
			}
		}
		input.IsLimitUp = isLimitUp
	}

	// 6a: Try LLM classification results (cache → DB) first.
	llmResults := s.loadLLMResults(ctx, date, candidateCodes)
	topicIDMap := make(map[string]int64)
	needHeatCodes := make([]string, 0)
	for _, tsCode := range candidateCodes {
		if id, ok := llmResults[tsCode]; ok && id != 0 {
			topicIDMap[tsCode] = id
		} else {
			needHeatCodes = append(needHeatCodes, tsCode)
		}
	}

	// 6b: Heat classification fills gaps for stocks not covered by LLM.
	if len(needHeatCodes) > 0 {
		heatInputs := make([]classify.StockQuoteInput, 0, len(needHeatCodes))
		for _, code := range needHeatCodes {
			for i := range allHeatInputs {
				if allHeatInputs[i].TsCode == code {
					heatInputs = append(heatInputs, allHeatInputs[i])
					break
				}
			}
		}
		heatResults, err := classifySvc.ClassifyBySimpleHeat(ctx, heatInputs, date)
		if err != nil {
			logger.Warn("heat classify snapshot stocks failed", zap.Error(err), zap.Int("count", len(heatInputs)))
		}
		for _, tsCode := range needHeatCodes {
			if topics, ok := heatResults[tsCode]; ok && len(topics) > 0 && topics[0].TopicID != 0 {
				topicIDMap[tsCode] = topics[0].TopicID
			}
		}
	}

	missingTopicCodes := make([]string, 0)
	for _, tsCode := range candidateCodes {
		if _, ok := topicIDMap[tsCode]; !ok {
			missingTopicCodes = append(missingTopicCodes, tsCode)
		}
	}
	for _, tsCode := range missingTopicCodes {
		logger.Warn("snapshot stock has no classified topic",
			zap.String("date", date),
			zap.String("ts_code", tsCode),
		)
	}

	// Step 7: Get yesterday's snapshot for consecutive_strong_days calculation
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t, _ := time.ParseInLocation("2006-01-02", date, loc)
	yesterday := utils.PreviousTradingDay(t)
	yesterdaySnapshots, _ := snapshotRepo.GetByDate(ctx, yesterday.Format("2006-01-02"))
	yesterdayMap := make(map[string]*dal_model.DailyStockSnapshot, len(yesterdaySnapshots))
	for i := range yesterdaySnapshots {
		yesterdayMap[yesterdaySnapshots[i].TsCode] = &yesterdaySnapshots[i]
	}

	// Step 8: Build snapshot records
	records := make([]dal_model.DailyStockSnapshot, 0, len(candidateCodes))
	for _, tsCode := range candidateCodes {
		quote := quoteMap[tsCode]
		name := stockNameMap[tsCode]
		if name == "" {
			name = tsCode
		}

		// Determine if limit-up via LimitListD or limiter.DetectLimitUp
		isLimitUp := false
		var limitTimes int16
		if limitItem, ok := limitMap[tsCode]; ok {
			isLimitUp = true
			limitTimes = int16(limitItem.LimitTimes)
		} else {
			board := limiter.DetectBoard(tsCode[:6])
			rule := boardRules[board]
			output := limiter.DetectLimitUp(dal_model.DetectInput{
				TsCode:       tsCode,
				StockName:    name,
				CurrentPrice: quote.Price,
				PreClose:     quote.PreClose,
				ChangePct:    quote.PctChg,
				BoardCode:    board,
			}, rule)
			isLimitUp = output.IsLimitUp
		}

		// Calculate consecutive strong days
		var consecutiveDays int16 = 1
		if prev, ok := yesterdayMap[tsCode]; ok {
			consecutiveDays = prev.ConsecutiveStrongDays + 1
		}

		// Market value
		var totalMv *float64
		if mv, ok := mvMap[tsCode]; ok && mv > 0 {
			totalMv = &mv
		}

		changePct := quote.PctChg
		vol := float64(quote.Vol)
		amount := quote.Amount
		var topicID *int64
		if classifiedTopicID, ok := topicIDMap[tsCode]; ok {
			topicID = &classifiedTopicID
		}

		records = append(records, buildSnapshotRecord(t, tsCode, name, topicID, changePct, isLimitUp, limitTimes, consecutiveDays, totalMv, vol, amount))

	}

	// Step 9: Replace existing records only after the new snapshot is fully prepared.
	if err := snapshotRepo.DeleteByDate(ctx, date); err != nil {
		return fmt.Errorf("delete existing snapshot: %w", err)
	}

	// Step 10: Batch insert
	if err := snapshotRepo.InsertBatch(ctx, records); err != nil {
		return fmt.Errorf("insert snapshot batch: %w", err)
	}

	logger.Info("snapshot completed",
		zap.String("date", date),
		zap.Int("total", len(records)))

	return nil
}

func (s *SnapshotServiceImpl) loadLLMResults(ctx context.Context, date string, tsCodes []string) map[string]int64 {
	if len(tsCodes) == 0 {
		return nil
	}

	// 1. Try in-memory cache first.
	llmCache := cache.NewLLMClassificationCache()
	if cached, hit, _ := llmCache.GetClassificationResult(ctx, date); hit && len(cached) > 0 {
		result := make(map[string]int64)
		for _, tsCode := range tsCodes {
			if rels, ok := cached[tsCode]; ok && len(rels) > 0 && rels[0].TopicID != 0 {
				result[tsCode] = rels[0].TopicID
			}
		}
		if len(result) > 0 {
			logger.Info("snapshot: loaded llm results from cache", zap.Int("count", len(result)))
			return result
		}
	}

	// 2. Fallback to database.
	daoImpl := dao.NewLLMClassifyResultDAO()
	rows, err := daoImpl.GetByDateAndTsCodes(ctx, date, tsCodes)
	if err != nil || len(rows) == 0 {
		return nil
	}

	topicNames := make([]string, 0, len(rows))
	for _, row := range rows {
		topicNames = append(topicNames, row.Topic)
	}

	topicRepo := repo.NewTopicRepository()
	topicList, err := topicRepo.GetByNames(ctx, topicNames)
	if err != nil {
		logger.Warn("snapshot: lookup topics by names failed", zap.Error(err))
		return nil
	}
	nameToID := make(map[string]int64, len(topicList))
	for _, t := range topicList {
		nameToID[t.Name] = t.ID
	}

	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		if id, ok := nameToID[row.Topic]; ok {
			result[row.TsCode] = id
		}
	}
	if len(result) > 0 {
		logger.Info("snapshot: loaded llm results from db", zap.Int("count", len(result)))
	}
	return result
}
