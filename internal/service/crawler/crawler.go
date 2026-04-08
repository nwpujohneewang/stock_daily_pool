package crawler

import (
	"context"
	"fmt"
	"regexp"
	"stock/model/dal_model"
	"time"

	"stock/dal/dao"
	"stock/dal/repo"
	"stock/internal/external/jiuyan"
	"stock/internal/pkg/converter"
	"stock/internal/pkg/logger"
	"stock/internal/pkg/utils"

	"go.uber.org/zap"
)

type CrawlerServiceImpl struct{}

type relationKey struct {
	tsCode  string
	topicID int64
}

type dateRange struct {
	min time.Time
	max time.Time
}

type aggregatedRelationRow struct {
	TsCode    string
	TopicID   int64
	TopicName string
	Category  string
	Source    string
	HitCount  int
	FirstSeen time.Time
	LastSeen  time.Time
}

var computeTopicStockCountAfterRebuild = func(ctx context.Context) error {
	topicStockCountRepo := repo.NewTopicStockCountRepository()
	counts, err := topicStockCountRepo.AggregateFromRelations(ctx)
	if err != nil {
		return err
	}
	return topicStockCountRepo.UpsertBatch(ctx, counts)
}

func updateDateRange(r *dateRange, t time.Time) {
	if t.IsZero() {
		return
	}
	if r.min.IsZero() || t.Before(r.min) {
		r.min = t
	}
	if r.max.IsZero() || t.After(r.max) {
		r.max = t
	}
}

func NewCrawlerService() *CrawlerServiceImpl {
	return &CrawlerServiceImpl{}
}

func mergeNormalizedRelations(relations []dal_model.StockTopicRelation) []dal_model.StockTopicRelation {
	if len(relations) == 0 {
		return nil
	}

	merged := make(map[relationKey]*dal_model.StockTopicRelation, len(relations))
	order := make([]relationKey, 0, len(relations))
	for _, relation := range relations {
		key := relationKey{tsCode: relation.TsCode, topicID: relation.TopicID}
		existing := merged[key]
		if existing == nil {
			copied := relation
			merged[key] = &copied
			order = append(order, key)
			continue
		}

		existing.HitCount += relation.HitCount
		if relation.FirstSeenDate != nil && (existing.FirstSeenDate == nil || relation.FirstSeenDate.Before(*existing.FirstSeenDate)) {
			firstSeen := *relation.FirstSeenDate
			existing.FirstSeenDate = &firstSeen
		}
		if relation.LastSeenDate != nil && (existing.LastSeenDate == nil || relation.LastSeenDate.After(*existing.LastSeenDate)) {
			lastSeen := *relation.LastSeenDate
			existing.LastSeenDate = &lastSeen
		}
	}

	result := make([]dal_model.StockTopicRelation, 0, len(order))
	for _, key := range order {
		result = append(result, *merged[key])
	}
	return result
}

func buildRebuildRelations(rows []aggregatedRelationRow) []dal_model.StockTopicRelation {
	relations := make([]dal_model.StockTopicRelation, 0, len(rows))
	for _, row := range rows {
		firstSeen := row.FirstSeen
		lastSeen := row.LastSeen
		relations = append(relations, dal_model.StockTopicRelation{
			TsCode:        row.TsCode,
			TopicID:       row.TopicID,
			TopicName:     row.TopicName,
			Category:      row.Category,
			Source:        row.Source,
			HitCount:      row.HitCount,
			FirstSeenDate: &firstSeen,
			LastSeenDate:  &lastSeen,
		})
	}
	return relations
}

func mergeOverwriteRelations(existing map[relationKey]dal_model.StockTopicRelation, rebuild []dal_model.StockTopicRelation) []dal_model.StockTopicRelation {
	if len(rebuild) == 0 {
		return nil
	}

	result := make([]dal_model.StockTopicRelation, 0, len(rebuild))
	for _, relation := range rebuild {
		key := relationKey{tsCode: relation.TsCode, topicID: relation.TopicID}
		if current, ok := existing[key]; ok {
			current.TopicName = relation.TopicName
			current.Category = relation.Category
			current.Source = relation.Source
			current.HitCount = relation.HitCount
			current.FirstSeenDate = relation.FirstSeenDate
			current.LastSeenDate = relation.LastSeenDate
			result = append(result, current)
			continue
		}
		result = append(result, relation)
	}
	return result
}

func refreshTopicStockCountAfterRebuild(ctx context.Context) error {
	return computeTopicStockCountAfterRebuild(ctx)
}

func (s *CrawlerServiceImpl) CrawlDate(ctx context.Context, date string) error {
	data, err := jiuyan.FetchFieldData(ctx, date)
	if err != nil {
		return fmt.Errorf("fetch jiuyan data: %w", err)
	}

	topicRepo := repo.NewTopicRepository()
	mappingRepo := repo.NewStockTopicRelationRepository()
	topicDictRepo := repo.NewTopicDictionaryRepository()

	topicDictMap, err := topicDictRepo.GetAllMap(ctx)
	if err != nil {
		logger.Warn("load topic_dictionary failed, using raw topic names", zap.Error(err))
		topicDictMap = make(map[string]dal_model.TopicDictionary)
	}

	now := dao.Now()
	topicMap := make(map[string]*dal_model.Topic, len(data))
	for _, field := range data {
		topicName := field.Name

		if dict, ok := topicDictMap[field.Name]; ok {
			topicName = dict.NormalizedName
		}

		if _, exists := topicMap[topicName]; !exists {
			topic := &dal_model.Topic{
				Name:          topicName,
				Source:        "jiuyan",
				JiuyanFieldID: &field.ActionFieldID,
			}
			topicMap[topicName] = topic
		}
	}

	topics := make([]dal_model.Topic, 0, len(topicMap))
	topicNames := make([]string, 0, len(topicMap))
	for name, topic := range topicMap {
		topics = append(topics, *topic)
		topicNames = append(topicNames, name)
	}

	if err = topicRepo.UpsertBatch(ctx, topics); err != nil {
		logger.Warn("upsert topic batch failed", zap.Error(err))
		return nil
	}

	topicRows, err := topicRepo.GetByNames(ctx, topicNames)
	if err != nil {
		return nil
	}
	topicIDMap := make(map[string]int64, len(topicRows))
	for _, t := range topicRows {
		topicIDMap[t.Name] = t.ID
	}

	relations := make([]dal_model.StockTopicRelation, 0)
	for _, field := range data {
		topicName := field.Name
		if dict, ok := topicDictMap[field.Name]; ok {
			topicName = dict.NormalizedName
		}
		topicID := topicIDMap[topicName]
		if topicID == 0 {
			continue
		}
		for _, stock := range field.List {
			tsCode := converter.JiuyanToTushare(stock.Code)
			if tsCode == "" {
				continue
			}
			relations = append(relations, dal_model.StockTopicRelation{
				TsCode:        tsCode,
				TopicID:       topicID,
				Source:        "jiuyan",
				HitCount:      1,
				LastSeenDate:  &now,
				FirstSeenDate: &now,
			})
		}
	}
	if err := mappingRepo.UpsertBatch(ctx, relations); err != nil {
		logger.Warn("upsert mapping batch failed", zap.Error(err))
	}
	return nil
}

func (s *CrawlerServiceImpl) CrawlHistory(ctx context.Context, startDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("parse start date: %w", err)
	}

	for d := start; d.Before(dao.Now()); d = d.AddDate(0, 0, 1) {
		if !utils.IsTradingDay(d) {
			continue
		}
		dateStr := d.Format("2006-01-02")
		if err := s.CrawlDate(ctx, dateStr); err != nil {
			logger.Warn("crawl date failed", zap.String("date", dateStr), zap.Error(err))
		}
	}
	return nil
}

func (s *CrawlerServiceImpl) CrawlToday(ctx context.Context) error {
	yesterday := utils.PreviousTradingDay(dao.Now())
	dateStr := yesterday.Format("2006-01-02")
	return s.CrawlDate(ctx, dateStr)
}

// ValidateTopics 检查 topic 是否都在词典中
func (s *CrawlerServiceImpl) ValidateTopics(ctx context.Context, topicNames []string) (missing []string, err error) {
	topicDictRepo := repo.NewTopicDictionaryRepository()
	dictMap, err := topicDictRepo.GetAllMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("load topic dictionary: %w", err)
	}

	for _, name := range topicNames {
		if _, ok := dictMap[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

// CrawlWithParams 使用指定凭证爬取数据
func (s *CrawlerServiceImpl) CrawlWithParams(ctx context.Context, params CrawlParams) (*CrawlResult, error) {
	// 验证日期格式
	if params.Date == "" {
		return nil, ErrEmptyDate
	}
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateRegex.MatchString(params.Date) {
		return nil, ErrInvalidDate
	}

	// 调用韭研 API
	data, err := jiuyan.FetchFieldDataWithParams(ctx, params.Date, params.Token, params.Cookie, params.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("fetch jiuyan data: %w", err)
	}

	// 先保存原始数据到 jiuyan_raw_data
	if err = s.saveRawData(ctx, params.Date, data); err != nil {
		logger.Error("save jiuyan raw data failed", zap.Error(err))
		return nil, err
	}

	// 收集所有 topic 名称进行验证
	topicNames := make([]string, 0, len(data))
	for _, field := range data {
		topicNames = append(topicNames, field.Name)
	}

	// 验证 topics
	missing, err := s.ValidateTopics(ctx, topicNames)
	if err != nil {
		return nil, fmt.Errorf("validate topics: %w", err)
	}
	if len(missing) > 0 {
		return nil, &MissingTopicsError{Topics: missing}
	}

	// 处理数据
	topicsCount, stocksCount, err := s.processFieldData(ctx, params.Date, data)
	if err != nil {
		return nil, err
	}

	return &CrawlResult{
		Date:        params.Date,
		TopicsCount: topicsCount,
		StocksCount: stocksCount,
	}, nil
}

// saveRawData 保存原始数据到 jiuyan_raw_data 表
func (s *CrawlerServiceImpl) saveRawData(ctx context.Context, date string, data []jiuyan.FieldData) error {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("parse date: %w", err)
	}

	rawDataRepo := repo.NewJiuyanRawDataRepository()
	records := make([]dal_model.JiuyanRawData, 0)

	for _, field := range data {
		for _, stock := range field.List {
			records = append(records, dal_model.JiuyanRawData{
				Date:          parsedDate,
				TopicName:     field.Name,
				ActionFieldID: field.ActionFieldID,
				StockCode:     stock.Code,
				StockName:     stock.Name,
			})
		}
	}

	if len(records) == 0 {
		return nil
	}

	if err := rawDataRepo.UpsertBatch(ctx, records); err != nil {
		return fmt.Errorf("upsert raw data batch: %w", err)
	}

	return nil
}

// processFieldData 处理韭研返回的数据
func (s *CrawlerServiceImpl) processFieldData(ctx context.Context, date string, data []jiuyan.FieldData) (topicsCount, stocksCount int, err error) {
	topicRepo := repo.NewTopicRepository()
	mappingRepo := repo.NewStockTopicRelationRepository()
	topicDictRepo := repo.NewTopicDictionaryRepository()

	dictMap, err := topicDictRepo.GetAllMap(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("load topic dictionary: %w", err)
	}

	// 解析日期
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, 0, fmt.Errorf("parse date: %w", err)
	}

	stockSet := make(map[string]struct{})
	topicMap := make(map[string]*dal_model.Topic, len(data))
	for _, field := range data {
		// 获取标准化名称
		normalizedName := field.Name
		if dict, ok := dictMap[field.Name]; ok {
			normalizedName = dict.NormalizedName
		}

		if _, exists := topicMap[normalizedName]; !exists {
			category := ""
			if dict, ok := dictMap[field.Name]; ok {
				category = dict.Category
			}
			topic := &dal_model.Topic{
				Name:          normalizedName,
				Category:      category,
				Source:        "jiuyan",
				JiuyanFieldID: &field.ActionFieldID,
			}
			topicMap[normalizedName] = topic
		}
	}

	topics := make([]dal_model.Topic, 0, len(topicMap))
	topicNames := make([]string, 0, len(topicMap))
	for name, topic := range topicMap {
		topics = append(topics, *topic)
		topicNames = append(topicNames, name)
	}
	if err = topicRepo.UpsertBatch(ctx, topics); err != nil {
		return 0, 0, fmt.Errorf("upsert topic batch failed: %w", err)
	}
	topicsCount = len(topicMap)

	topicRows, err := topicRepo.GetByNames(ctx, topicNames)
	if err != nil {
		return 0, 0, fmt.Errorf("get topic by names failed: %w", err)
	}
	topicIDMap := make(map[string]int64, len(topicRows))
	for _, t := range topicRows {
		topicIDMap[t.Name] = t.ID
	}

	relations := make([]dal_model.StockTopicRelation, 0)
	for _, field := range data {
		normalizedName := field.Name
		if dict, ok := dictMap[field.Name]; ok {
			normalizedName = dict.NormalizedName
		}
		topicID := topicIDMap[normalizedName]
		if topicID == 0 {
			continue
		}
		category := ""
		if dict, ok := dictMap[field.Name]; ok {
			category = dict.Category
		}
		for _, stock := range field.List {
			tsCode := converter.JiuyanToTushare(stock.Code)
			if tsCode == "" {
				continue
			}
			relations = append(relations, dal_model.StockTopicRelation{
				TsCode:        tsCode,
				TopicID:       topicID,
				TopicName:     normalizedName,
				Category:      category,
				Source:        "jiuyan",
				HitCount:      1,
				LastSeenDate:  &parsedDate,
				FirstSeenDate: &parsedDate,
			})
			stockSet[tsCode] = struct{}{}
		}
	}

	if err := mappingRepo.BulkUpsertAccumulate(ctx, relations); err != nil {
		return 0, 0, fmt.Errorf("upsert relation batch failed: %w", err)
	}

	stocksCount = len(stockSet)
	return topicsCount, stocksCount, nil
}

// RebuildTopicRelations 从 jiuyan_raw_data 重建指定日期范围的 topic relations
func (s *CrawlerServiceImpl) RebuildTopicRelations(ctx context.Context, startDate, endDate string) (*RebuildResult, error) {
	// 1. 校验日期格式
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateRegex.MatchString(startDate) || !dateRegex.MatchString(endDate) {
		return nil, ErrInvalidDate
	}

	// 2. 校验日期范围
	if startDate > endDate {
		return nil, ErrInvalidDateRange
	}

	// 3. 查询日期范围内的所有唯一 topic_name
	rawDataRepo := repo.NewJiuyanRawDataRepository()
	topicNames, err := rawDataRepo.GetDistinctTopicNames(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query jiuyan_raw_data: %w", err)
	}

	if len(topicNames) == 0 {
		return &RebuildResult{
			StartDate:      startDate,
			EndDate:        endDate,
			TopicsCount:    0,
			RelationsCount: 0,
			Topics:         []RebuildTopicInfo{},
		}, nil
	}

	// 4. 加载 topic_dictionary
	topicDictRepo := repo.NewTopicDictionaryRepository()
	dictMap, err := topicDictRepo.GetAllMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("load topic dictionary: %w", err)
	}

	// 5. 检查缺失映射
	var missing []string
	for _, name := range topicNames {
		if _, ok := dictMap[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, &MissingTopicsError{Topics: missing}
	}

	// 6. 查询聚合数据
	rows, err := rawDataRepo.GetAggregatedData(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query aggregated raw data: %w", err)
	}

	// 7. 批量处理
	topicRepo := repo.NewTopicRepository()
	mappingRepo := repo.NewStockTopicRelationRepository()

	// 构建 topic -> stocks 映射用于统计，同时收集 unique topics
	topicStocksMap := make(map[string]map[string]struct{})
	topicMap := make(map[string]*dal_model.Topic)

	for _, r := range rows {
		dict := dictMap[r.TopicName]
		normalizedName := dict.NormalizedName

		// 统计每个 raw topic 下的 stocks
		if _, ok := topicStocksMap[r.TopicName]; !ok {
			topicStocksMap[r.TopicName] = make(map[string]struct{})
		}
		topicStocksMap[r.TopicName][r.StockCode] = struct{}{}

		// 收集 unique topics
		if _, exists := topicMap[normalizedName]; !exists {
			topicMap[normalizedName] = &dal_model.Topic{
				Name:     normalizedName,
				Source:   "jiuyan",
				Category: dict.Category,
			}
		}
	}

	// 批量 upsert topics
	topics := make([]dal_model.Topic, 0, len(topicMap))
	topicNameList := make([]string, 0, len(topicMap))
	for name, topic := range topicMap {
		topics = append(topics, *topic)
		topicNameList = append(topicNameList, name)
	}

	if err = topicRepo.UpsertBatch(ctx, topics); err != nil {
		return nil, fmt.Errorf("batch upsert topics failed: %w", err)
	}

	// 批量获取 topic IDs
	topicRows, err := topicRepo.GetByNames(ctx, topicNameList)
	if err != nil {
		return nil, fmt.Errorf("batch get topic ids failed: %w", err)
	}
	topicIDMap := make(map[string]int64, len(topicRows))
	for _, t := range topicRows {
		topicIDMap[t.Name] = t.ID
	}

	// 构建结果
	result := &RebuildResult{
		StartDate:   startDate,
		EndDate:     endDate,
		TopicsCount: len(topicMap),
		Topics:      make([]RebuildTopicInfo, 0, len(topicNames)),
	}

	// 构建 RebuildTopicInfo 列表（按 raw topic name）
	processedRawTopics := make(map[string]bool)
	for _, r := range rows {
		if !processedRawTopics[r.TopicName] {
			processedRawTopics[r.TopicName] = true
			dict := dictMap[r.TopicName]
			result.Topics = append(result.Topics, RebuildTopicInfo{
				Name:           r.TopicName,
				NormalizedName: dict.NormalizedName,
				StocksCount:    len(topicStocksMap[r.TopicName]),
			})
		}
	}

	rowFirstSeenRange := &dateRange{}
	rowLastSeenRange := &dateRange{}
	for _, r := range rows {
		updateDateRange(rowFirstSeenRange, r.FirstSeen)
		updateDateRange(rowLastSeenRange, r.LastSeen)
	}

	// 收集所有 relations
	relationRows := make([]aggregatedRelationRow, 0, len(rows))
	for _, r := range rows {
		dict := dictMap[r.TopicName]
		normalizedName := dict.NormalizedName
		topicID := topicIDMap[normalizedName]
		if topicID == 0 {
			continue
		}

		tsCode := converter.JiuyanToTushare(r.StockCode)
		if tsCode == "" {
			continue
		}

		relationRows = append(relationRows, aggregatedRelationRow{
			TsCode:    tsCode,
			TopicID:   topicID,
			TopicName: normalizedName,
			Category:  dict.Category,
			Source:    "jiuyan",
			HitCount:  r.HitCount,
			FirstSeen: r.FirstSeen,
			LastSeen:  r.LastSeen,
		})
	}
	relations := buildRebuildRelations(relationRows)
	relations = mergeNormalizedRelations(relations)

	relationFirstSeenRange := &dateRange{}
	relationLastSeenRange := &dateRange{}
	for _, relation := range relations {
		if relation.FirstSeenDate != nil {
			updateDateRange(relationFirstSeenRange, *relation.FirstSeenDate)
		}
		if relation.LastSeenDate != nil {
			updateDateRange(relationLastSeenRange, *relation.LastSeenDate)
		}
	}

	// 批量 upsert relations
	if err = mappingRepo.BulkUpsertReplace(ctx, relations); err != nil {
		return nil, fmt.Errorf("batch upsert relations failed: %w", err)
	}
	if err = refreshTopicStockCountAfterRebuild(ctx); err != nil {
		return nil, fmt.Errorf("recompute topic stock count failed: %w", err)
	}
	result.RelationsCount = len(relations)

	logger.Info("rebuild topic relations completed",
		zap.Int("topics_count", result.TopicsCount),
		zap.Int("relations_count", result.RelationsCount))

	return result, nil
}
