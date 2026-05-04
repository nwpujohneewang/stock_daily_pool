// internal/service/pool_service.go
package pool

import (
	"context"
	"stock/config"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/external/tushare"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"
	"stock/internal/pkg/utils"
	"stock/model/dal_model"
	"strings"
	"time"

	"go.uber.org/zap"
)

type PoolServiceImpl struct{}

func NewPoolService() *PoolServiceImpl {
	return &PoolServiceImpl{}
}

func (s *PoolServiceImpl) GetLimitUp(ctx context.Context, params PoolQueryParams) ([]PoolItemResult, error) {
	pools, err := repo.NewPoolRepository().GetByDate(ctx, params.Date, 1)
	if err != nil {
		return nil, err
	}

	results := make([]PoolItemResult, 0, len(pools))
	for _, p := range pools {
		result := PoolItemResult{
			TsCode:   p.TsCode,
			Name:     p.StockName,
			PoolType: p.PoolType,
			Date:     p.Date.Format("2006-01-02"),
		}
		if p.CurrentPrice != nil {
			result.Price = *p.CurrentPrice
		}
		if p.PreClose != nil {
			result.PreClose = *p.PreClose
		}
		if p.ChangePct != nil {
			result.ChangePct = *p.ChangePct
		}
		if p.LimitUpPrice != nil {
			result.LimitUpPrice = *p.LimitUpPrice
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *PoolServiceImpl) GetAbove5(ctx context.Context, params PoolQueryParams) ([]PoolItemResult, error) {
	pools, err := repo.NewPoolRepository().GetByDate(ctx, params.Date, 2)
	if err != nil {
		return nil, err
	}

	results := make([]PoolItemResult, 0, len(pools))
	for _, p := range pools {
		result := PoolItemResult{
			TsCode:   p.TsCode,
			Name:     p.StockName,
			PoolType: p.PoolType,
			Date:     p.Date.Format("2006-01-02"),
		}
		if p.CurrentPrice != nil {
			result.Price = *p.CurrentPrice
		}
		if p.PreClose != nil {
			result.PreClose = *p.PreClose
		}
		if p.ChangePct != nil {
			result.ChangePct = *p.ChangePct
		}
		if p.LimitUpPrice != nil {
			result.LimitUpPrice = *p.LimitUpPrice
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *PoolServiceImpl) Reclassify(ctx context.Context, date string) (*ReclassifyResult, error) {
	if s.isToday(date) {
		return s.reclassifyToday(ctx, date)
	}
	return s.reclassifyHistorical(ctx, date)
}

func (s *PoolServiceImpl) isCurrentlyTrading() bool {
	now := time.Now().UTC()
	nowBJ := now.Add(8 * time.Hour)

	weekday := nowBJ.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hour := nowBJ.Hour()
	minute := nowBJ.Minute()
	timeVal := hour*100 + minute

	if (timeVal >= 915 && timeVal <= 1135) || (timeVal >= 1300 && timeVal <= 1510) {
		return true
	}
	return false
}

func (s *PoolServiceImpl) buildYesterdayStrongItems(
	prevSnapshots []dal_model.DailyStockSnapshot,
	quoteMap map[string]*dal_model.StockQuote,
) []ReclassifyItemResult {
	items := make([]ReclassifyItemResult, 0, len(prevSnapshots))
	for _, snap := range prevSnapshots {
		if snap.ChangePct == nil {
			continue
		}

		quote := quoteMap[snap.TsCode]
		if quote == nil {
			continue
		}

		vol := float64(quote.Vol)
		amount := quote.Amount
		item := ReclassifyItemResult{
			TsCode:                snap.TsCode,
			Name:                  snap.StockName,
			Price:                 quote.Price,
			PreClose:              quote.PreClose,
			ChangePct:             quote.PctChg,
			YesterdayChangePct:    *snap.ChangePct,
			IsYesterdayStrong:     true,
			BoardCode:             string(limiter.DetectBoard(snap.TsCode[:6])),
			PoolType:              dal_model.PoolTypeYesterdayStrong,
			ConsecutiveStrongDays: snap.ConsecutiveStrongDays,
			LimitTimes:            snap.LimitTimes,
			TotalMv:               snap.TotalMv,
			Vol:                   &vol,
			Amount:                &amount,
		}
		if quote.PctChg >= 5.0 {
			item.ConsecutiveStrongDays = snap.ConsecutiveStrongDays + 1
		}
		items = append(items, item)
	}
	return items
}

func (s *PoolServiceImpl) buildHistoricalYesterdayStrongItems(
	prevSnapshots []dal_model.DailyStockSnapshot,
	quoteMap map[string]*tushare.QuoteItem,
) []ReclassifyItemResult {
	items := make([]ReclassifyItemResult, 0, len(prevSnapshots))
	for _, snap := range prevSnapshots {
		if snap.ChangePct == nil {
			continue
		}

		quote := quoteMap[snap.TsCode]
		if quote == nil {
			continue
		}

		item := ReclassifyItemResult{
			TsCode:                snap.TsCode,
			Name:                  snap.StockName,
			Price:                 quote.Price,
			PreClose:              quote.PreClose,
			ChangePct:             quote.PctChg,
			YesterdayChangePct:    *snap.ChangePct,
			IsYesterdayStrong:     true,
			BoardCode:             string(limiter.DetectBoard(snap.TsCode[:6])),
			PoolType:              dal_model.PoolTypeYesterdayStrong,
			ConsecutiveStrongDays: snap.ConsecutiveStrongDays,
			LimitTimes:            snap.LimitTimes,
			TotalMv:               snap.TotalMv,
		}
		if quote.PctChg >= 5.0 {
			item.ConsecutiveStrongDays = snap.ConsecutiveStrongDays + 1
		}
		if quote.Vol > 0 {
			vol := float64(quote.Vol)
			item.Vol = &vol
		}
		if quote.Amount > 0 {
			amount := quote.Amount
			item.Amount = &amount
		}
		items = append(items, item)
	}
	return items
}

func buildSnapshotTopicMap(topics []dal_model.Topic) map[int64]dal_model.Topic {
	result := make(map[int64]dal_model.Topic, len(topics))
	for _, topic := range topics {
		result[topic.ID] = topic
	}
	return result
}

func buildSnapshotTopicRelation(topicID *int64, topicMap map[int64]dal_model.Topic) ([]dal_model.TopicRelation, error) {
	if topicID == nil {
		return nil, nil
	}
	topic, ok := topicMap[*topicID]
	if !ok {
		return nil, ErrHistoricalSnapshotNotGenerated
	}
	return []dal_model.TopicRelation{
		{
			TopicID:   topic.ID,
			TopicName: topic.Name,
			Category:  topic.Category,
			Source:    topic.Source,
		},
	}, nil
}

func getPreviousTradingDate(date string) (string, error) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return "", err
	}
	prev := utils.PreviousTradingDay(t)
	return prev.Format("2006-01-02"), nil
}

func (s *PoolServiceImpl) isToday(date string) bool {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	today := time.Now().In(loc).Format("20060102")
	normalizedDate := strings.ReplaceAll(date, "-", "")
	return normalizedDate == today
}

func (s *PoolServiceImpl) reclassifyToday(ctx context.Context, date string) (*ReclassifyResult, error) {
	poolCache := cache.NewPoolCache()
	quoteCache := cache.NewQuoteCache()
	stockRepo := repo.NewStockRepository()
	classificationCache := cache.NewClassificationCache()
	snapshotRepo := repo.NewSnapshotRepository()

	// Get pool members from cache
	limitUpCodes, _ := poolCache.GetLimitUpMembers(ctx, date)
	above5Codes, _ := poolCache.GetAbove5Members(ctx, date)
	logger.Info("reclassify today: pool cache loaded",
		zap.String("date", date),
		zap.Int("limit_up_count", len(limitUpCodes)),
		zap.Int("above5_count", len(above5Codes)),
	)

	// Get classification results from cache
	classifyResults, exists, err := classificationCache.GetClassificationResult(ctx, date)
	if err != nil {
		return nil, err
	}
	if !exists || len(classifyResults) == 0 {
		return nil, ErrDataNotReady
	}

	// Build code set
	codeSet := make(map[string]struct{}, len(limitUpCodes)+len(above5Codes))
	for _, tsCode := range limitUpCodes {
		codeSet[tsCode] = struct{}{}
	}
	for _, tsCode := range above5Codes {
		codeSet[tsCode] = struct{}{}
	}

	if len(codeSet) == 0 {
		return nil, ErrDataNotReady
	}

	allCodes := make([]string, 0, len(codeSet))
	for tsCode := range codeSet {
		allCodes = append(allCodes, tsCode)
	}

	// Load yesterday's snapshot for consecutive_strong_days and limit_times,
	// and include them in quote loading so yesterday-strong items don't rely on a separate cache path.
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t, _ := time.ParseInLocation("2006-01-02", date, loc)
	yesterday := utils.PreviousTradingDay(t)
	yesterdayDate := yesterday.Format("2006-01-02")
	yesterdaySnapshots, _ := snapshotRepo.GetByDate(ctx, yesterdayDate)
	yesterdayMap := make(map[string]*dal_model.DailyStockSnapshot, len(yesterdaySnapshots))
	allCodeSet := make(map[string]struct{}, len(allCodes)+len(yesterdaySnapshots))
	for _, tsCode := range allCodes {
		allCodeSet[tsCode] = struct{}{}
	}
	for i := range yesterdaySnapshots {
		yesterdayMap[yesterdaySnapshots[i].TsCode] = &yesterdaySnapshots[i]
		allCodeSet[yesterdaySnapshots[i].TsCode] = struct{}{}
	}
	allQuoteCodes := make([]string, 0, len(allCodeSet))
	for tsCode := range allCodeSet {
		allQuoteCodes = append(allQuoteCodes, tsCode)
	}

	// Get stock info and quotes
	stocks, _ := stockRepo.GetByTsCodes(ctx, allCodes)
	stockMap := make(map[string]*dal_model.StockBasicInfo, len(stocks))
	for i := range stocks {
		stockMap[stocks[i].TsCode] = &stocks[i]
	}
	quoteMap, _ := quoteCache.GetBatch(ctx, allQuoteCodes)

	// Build items for limit-up stocks
	var items []ReclassifyItemResult
	for _, tsCode := range limitUpCodes {
		stock := stockMap[tsCode]
		quote := quoteMap[tsCode]
		if stock == nil || quote == nil {
			continue
		}

		vol := float64(quote.Vol / 1000)
		amount := quote.Amount / 1000.0
		// 实时总市值（万元）≈ 昨日总市值（万元） * (当前价 / 昨收)
		var mv *float64
		if stock.TotalMv != nil {
			if quote.PreClose > 0 {
				val := *stock.TotalMv * (quote.Price / quote.PreClose)
				mv = &val
			} else {
				// 兜底：若缺昨收，则用涨跌幅估算(1 + pct/100)
				val := *stock.TotalMv * (1.0 + quote.PctChg/100.0)
				mv = &val
			}
		}
		item := ReclassifyItemResult{
			TsCode:      tsCode,
			Name:        stock.Name,
			Price:       quote.Price,
			PreClose:    quote.PreClose,
			ChangePct:   quote.PctChg,
			BoardCode:   string(limiter.DetectBoard(tsCode[:6])),
			IsLimitUp:   true,
			IsAbove5Pct: true,
			PoolType:    1,
			TotalMv:     mv,
			Vol:         &vol,
			Amount:      &amount,
		}

		// Compute limit_times from yesterday snapshot
		if prev, ok := yesterdayMap[tsCode]; ok && prev.IsLimitUp {
			item.LimitTimes = prev.LimitTimes + 1
		} else {
			item.LimitTimes = 1
		}

		// Compute consecutive_strong_days
		if prev, ok := yesterdayMap[tsCode]; ok {
			item.ConsecutiveStrongDays = prev.ConsecutiveStrongDays + 1
		} else {
			item.ConsecutiveStrongDays = 1
		}

		if topics, ok := classifyResults[tsCode]; ok && len(topics) > 0 {
			item.Topics = topics
		}
		items = append(items, item)
	}

	// Build items for above5 stocks
	for _, tsCode := range above5Codes {
		stock := stockMap[tsCode]
		quote := quoteMap[tsCode]
		if stock == nil || quote == nil {
			continue
		}

		vol := float64(quote.Vol / 1000)
		amount := quote.Amount / 1000.0
		// 实时总市值（万元）≈ 昨日总市值（万元） * (当前价 / 昨收)
		var mv *float64
		if stock.TotalMv != nil {
			if quote.PreClose > 0 {
				val := (*stock.TotalMv) * (quote.Price / quote.PreClose)
				mv = &val
			} else {
				val := (*stock.TotalMv) * (1.0 + quote.PctChg/100.0)
				mv = &val
			}
		}
		item := ReclassifyItemResult{
			TsCode:      tsCode,
			Name:        stock.Name,
			Price:       quote.Price,
			PreClose:    quote.PreClose,
			ChangePct:   quote.PctChg,
			BoardCode:   string(limiter.DetectBoard(tsCode[:6])),
			IsLimitUp:   false,
			IsAbove5Pct: true,
			PoolType:    2,
			TotalMv:     mv,
			Vol:         &vol,
			Amount:      &amount,
		}

		// Compute consecutive_strong_days
		if prev, ok := yesterdayMap[tsCode]; ok {
			item.ConsecutiveStrongDays = prev.ConsecutiveStrongDays + 1
		} else {
			item.ConsecutiveStrongDays = 1
		}

		if topics, ok := classifyResults[tsCode]; ok && len(topics) > 0 {
			item.Topics = topics
		}
		items = append(items, item)
	}

	isTrading := s.isCurrentlyTrading()
	yesterdayStrongItems := s.buildYesterdayStrongItems(yesterdaySnapshots, quoteMap)

	return &ReclassifyResult{
		Items:                items,
		YesterdayStrongItems: yesterdayStrongItems,
		IsTrading:            isTrading,
	}, nil
}

func (s *PoolServiceImpl) reclassifyHistorical(ctx context.Context, date string) (*ReclassifyResult, error) {
	topicRepo := repo.NewTopicRepository()
	snapshotRepo := repo.NewSnapshotRepository()

	tradeDate := strings.ReplaceAll(date, "-", "")
	tushareClient := tushare.NewClient(&config.GlobalConfig.Tushare, config.GlobalConfig.Retry)
	quotes, err := tushareClient.DailyAll(ctx, tradeDate)
	if err != nil {
		return nil, err
	}

	quoteMap := make(map[string]*tushare.QuoteItem, len(quotes))
	for _, quote := range quotes {
		quoteMap[quote.TsCode] = quote
	}

	snapshots, _ := snapshotRepo.GetByDate(ctx, date)
	if len(snapshots) == 0 {
		return nil, ErrHistoricalSnapshotNotGenerated
	}
	topicIDs := make([]int64, 0, len(snapshots))
	for i := range snapshots {
		if snapshots[i].TopicID == nil {
			continue
		}
		topicIDs = append(topicIDs, *snapshots[i].TopicID)
	}
	topics, err := topicRepo.GetByIDs(ctx, topicIDs)
	if err != nil {
		return nil, err
	}
	topicMap := buildSnapshotTopicMap(topics)

	var items []ReclassifyItemResult
	for _, snap := range snapshots {
		quote := quoteMap[snap.TsCode]
		if quote == nil {
			continue
		}
		board := limiter.DetectBoard(snap.TsCode[:6])

		topics, err := buildSnapshotTopicRelation(snap.TopicID, topicMap)
		if err != nil {
			return nil, err
		}

		item := ReclassifyItemResult{
			TsCode:                snap.TsCode,
			Name:                  snap.StockName,
			Price:                 quote.Price,
			PreClose:              quote.PreClose,
			ChangePct:             quote.PctChg,
			BoardCode:             string(board),
			IsLimitUp:             snap.IsLimitUp,
			IsAbove5Pct:           true,
			Topics:                topics,
			ClassifyLayer:         "snapshot",
			Confidence:            1,
			ConsecutiveStrongDays: snap.ConsecutiveStrongDays,
			LimitTimes:            snap.LimitTimes,
			TotalMv:               snap.TotalMv,
			Vol:                   snap.Vol,
			Amount:                snap.Amount,
		}
		if snap.IsLimitUp {
			item.PoolType = 1
		} else {
			item.PoolType = 2
		}
		items = append(items, item)
	}

	isTrading := s.isCurrentlyTrading()
	prevDate, err := getPreviousTradingDate(date)
	if err != nil {
		return nil, err
	}
	prevSnapshots, _ := snapshotRepo.GetByDate(ctx, prevDate)
	yesterdayStrongItems := s.buildHistoricalYesterdayStrongItems(prevSnapshots, quoteMap)

	return &ReclassifyResult{
		Items:                items,
		YesterdayStrongItems: yesterdayStrongItems,
		IsTrading:            isTrading,
	}, nil
}
