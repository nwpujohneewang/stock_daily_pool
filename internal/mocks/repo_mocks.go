package mocks

import (
	"context"
	"stock/model/dal_model"
)

type MockStockRepo struct {
	Stocks map[string]*dal_model.StockBasicInfo
	Err    error
}

func NewMockStockRepo() *MockStockRepo {
	return &MockStockRepo{
		Stocks: make(map[string]*dal_model.StockBasicInfo),
	}
}

func (m *MockStockRepo) GetByTsCode(ctx context.Context, tsCode string) (*dal_model.StockBasicInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Stocks[tsCode], nil
}

func (m *MockStockRepo) GetActiveStocks(ctx context.Context) ([]dal_model.StockBasicInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var stocks []dal_model.StockBasicInfo
	for _, s := range m.Stocks {
		stocks = append(stocks, *s)
	}
	return stocks, nil
}

func (m *MockStockRepo) Upsert(ctx context.Context, stock *dal_model.StockBasicInfo) error {
	if m.Err != nil {
		return m.Err
	}
	m.Stocks[stock.TsCode] = stock
	return nil
}

func (m *MockStockRepo) GetByBoardCode(ctx context.Context, boardCode string) ([]dal_model.StockBasicInfo, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var stocks []dal_model.StockBasicInfo
	for _, s := range m.Stocks {
		if s.BoardCode == boardCode {
			stocks = append(stocks, *s)
		}
	}
	return stocks, nil
}

type MockTopicRepo struct {
	Topics map[int64]*dal_model.Topic
	Err    error
}

func NewMockTopicRepo() *MockTopicRepo {
	return &MockTopicRepo{
		Topics: make(map[int64]*dal_model.Topic),
	}
}

func (m *MockTopicRepo) List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.Topic, int64, error) {
	if m.Err != nil {
		return nil, 0, m.Err
	}
	var topics []dal_model.Topic
	for _, t := range m.Topics {
		if keyword == "" || t.Name == keyword {
			topics = append(topics, *t)
		}
	}
	return topics, int64(len(topics)), nil
}

func (m *MockTopicRepo) GetByID(ctx context.Context, id int64) (*dal_model.Topic, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Topics[id], nil
}

func (m *MockTopicRepo) GetByName(ctx context.Context, name string) (*dal_model.Topic, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	for _, t := range m.Topics {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, nil
}

func (m *MockTopicRepo) Create(ctx context.Context, name, source string) (*dal_model.Topic, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	id := int64(len(m.Topics) + 1)
	topic := &dal_model.Topic{
		ID:     id,
		Name:   name,
		Source: source,
	}
	m.Topics[id] = topic
	return topic, nil
}

func (m *MockTopicRepo) Update(ctx context.Context, id int64, name string, isActive *bool) error {
	if m.Err != nil {
		return m.Err
	}
	if m.Topics[id] != nil {
		m.Topics[id].Name = name
		if isActive != nil {
			m.Topics[id].IsActive = *isActive
		}
	}
	return nil
}

func (m *MockTopicRepo) Delete(ctx context.Context, id int64) error {
	if m.Err != nil {
		return m.Err
	}
	delete(m.Topics, id)
	return nil
}

type MockAlertRepo struct {
	Alerts map[int64]*dal_model.StrategyAlert
	Err    error
}

func NewMockAlertRepo() *MockAlertRepo {
	return &MockAlertRepo{
		Alerts: make(map[int64]*dal_model.StrategyAlert),
	}
}

func (m *MockAlertRepo) GetTodayAlerts(ctx context.Context, date string) ([]dal_model.StrategyAlert, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var alerts []dal_model.StrategyAlert
	for _, a := range m.Alerts {
		alerts = append(alerts, *a)
	}
	return alerts, nil
}

func (m *MockAlertRepo) GetByDateRange(ctx context.Context, startDate, endDate string, page, pageSize int) ([]dal_model.StrategyAlert, int64, error) {
	if m.Err != nil {
		return nil, 0, m.Err
	}
	var alerts []dal_model.StrategyAlert
	for _, a := range m.Alerts {
		alerts = append(alerts, *a)
	}
	return alerts, int64(len(alerts)), nil
}

func (m *MockAlertRepo) Create(ctx context.Context, alert *dal_model.StrategyAlert) (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	id := int64(len(m.Alerts) + 1)
	alert.ID = id
	m.Alerts[id] = alert
	return id, nil
}

func (m *MockAlertRepo) GetByID(ctx context.Context, id int64) (*dal_model.StrategyAlert, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Alerts[id], nil
}

type MockPoolRepo struct {
	Pools []dal_model.DailyStockPool
	Err   error
}

func NewMockPoolRepo() *MockPoolRepo {
	return &MockPoolRepo{
		Pools: make([]dal_model.DailyStockPool, 0),
	}
}

func (m *MockPoolRepo) GetByDate(ctx context.Context, date string, poolType int) ([]dal_model.DailyStockPool, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var pools []dal_model.DailyStockPool
	for _, p := range m.Pools {
		if p.Date.Format("2006-01-02") == date && (poolType == 0 || int(p.PoolType) == poolType) {
			pools = append(pools, p)
		}
	}
	return pools, nil
}

func (m *MockPoolRepo) Upsert(ctx context.Context, pool *dal_model.DailyStockPool) error {
	if m.Err != nil {
		return m.Err
	}
	m.Pools = append(m.Pools, *pool)
	return nil
}

func (m *MockPoolRepo) GetByTsCodeAndDate(ctx context.Context, tsCode, date string, poolType int) (*dal_model.DailyStockPool, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	for _, p := range m.Pools {
		if p.TsCode == tsCode && p.Date.Format("2006-01-02") == date && int(p.PoolType) == poolType {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *MockPoolRepo) GetTopicStats(ctx context.Context, date string) (map[int64]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	stats := make(map[int64]int)
	return stats, nil
}

type MockMappingRepo struct {
	Mappings map[int64][]dal_model.TopicRelation
	Err      error
}

func NewMockMappingRepo() *MockMappingRepo {
	return &MockMappingRepo{
		Mappings: make(map[int64][]dal_model.TopicRelation),
	}
}

func (m *MockMappingRepo) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.TopicRelation, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var result []dal_model.TopicRelation
	for _, mappings := range m.Mappings {
		result = append(result, mappings...)
	}
	return result, nil
}

func (m *MockMappingRepo) Upsert(ctx context.Context, mapping *dal_model.TopicRelation) error {
	if m.Err != nil {
		return m.Err
	}
	m.Mappings[mapping.TopicID] = append(m.Mappings[mapping.TopicID], *mapping)
	return nil
}

func (m *MockMappingRepo) Delete(ctx context.Context, tsCode string, topicID int64) error {
	if m.Err != nil {
		return m.Err
	}
	mappings := m.Mappings[topicID]
	for i, mp := range mappings {
		if mp.TopicID == topicID {
			mappings = append(mappings[:i], mappings[i+1:]...)
			break
		}
	}
	m.Mappings[topicID] = mappings
	return nil
}

func (m *MockMappingRepo) GetByTopicID(ctx context.Context, topicID int64) ([]dal_model.TopicRelation, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Mappings[topicID], nil
}

func (m *MockMappingRepo) GetAllStockTopics(ctx context.Context) (map[int64][]dal_model.TopicRelation, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Mappings, nil
}

type MockBoardRepo struct {
	Boards map[string]dal_model.BoardRule
	Err    error
}

func NewMockBoardRepo() *MockBoardRepo {
	return &MockBoardRepo{
		Boards: make(map[string]dal_model.BoardRule),
	}
}

func (m *MockBoardRepo) GetAll(ctx context.Context) ([]dal_model.BoardRule, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var boards []dal_model.BoardRule
	for _, b := range m.Boards {
		boards = append(boards, b)
	}
	return boards, nil
}

func (m *MockBoardRepo) GetByCode(ctx context.Context, code string) (*dal_model.BoardRule, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	rule, ok := m.Boards[code]
	if !ok {
		return nil, nil
	}
	return &rule, nil
}

func (m *MockBoardRepo) Upsert(ctx context.Context, rule *dal_model.BoardRule) error {
	if m.Err != nil {
		return m.Err
	}
	m.Boards[string(rule.BoardCode)] = *rule
	return nil
}

type MockSynonymRepo struct {
	Synonyms   map[int64][]dal_model.TopicSynonym
	SynonymMap map[string]*dal_model.TopicSynonym
	Err        error
}

func NewMockSynonymRepo() *MockSynonymRepo {
	return &MockSynonymRepo{
		Synonyms:   make(map[int64][]dal_model.TopicSynonym),
		SynonymMap: make(map[string]*dal_model.TopicSynonym),
	}
}

func (m *MockSynonymRepo) GetByTopicID(ctx context.Context, topicID int64) ([]dal_model.TopicSynonym, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Synonyms[topicID], nil
}

func (m *MockSynonymRepo) GetBySynonym(ctx context.Context, synonym string) (*dal_model.TopicSynonym, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.SynonymMap[synonym], nil
}

func (m *MockSynonymRepo) Create(ctx context.Context, topicID int64, synonym, source string) error {
	if m.Err != nil {
		return m.Err
	}
	s := dal_model.TopicSynonym{
		ID:      int64(len(m.SynonymMap) + 1),
		TopicID: topicID,
		Synonym: synonym,
		Source:  source,
	}
	m.SynonymMap[synonym] = &s
	m.Synonyms[topicID] = append(m.Synonyms[topicID], s)
	return nil
}

func (m *MockSynonymRepo) Delete(ctx context.Context, synonymID int64) error {
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func (m *MockSynonymRepo) GetAll(ctx context.Context) ([]dal_model.TopicSynonym, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var result []dal_model.TopicSynonym
	for _, syns := range m.Synonyms {
		result = append(result, syns...)
	}
	return result, nil
}

type MockEvidenceRepo struct {
	Evidences map[int64]*dal_model.ClassificationAuditLog
	Err       error
}

func NewMockEvidenceRepo() *MockEvidenceRepo {
	return &MockEvidenceRepo{
		Evidences: make(map[int64]*dal_model.ClassificationAuditLog),
	}
}

func (m *MockEvidenceRepo) Create(ctx context.Context, evidence *dal_model.ClassificationAuditLog) (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	id := int64(len(m.Evidences) + 1)
	evidence.ID = id
	m.Evidences[id] = evidence
	return id, nil
}

func (m *MockEvidenceRepo) GetByTsCode(ctx context.Context, tsCode, date string, page, pageSize int) ([]dal_model.ClassificationAuditLog, int64, error) {
	if m.Err != nil {
		return nil, 0, m.Err
	}
	var evs []dal_model.ClassificationAuditLog
	for _, e := range m.Evidences {
		if e.TsCode == tsCode {
			evs = append(evs, *e)
		}
	}
	return evs, int64(len(evs)), nil
}

func (m *MockEvidenceRepo) GetByID(ctx context.Context, id int64) (*dal_model.ClassificationAuditLog, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Evidences[id], nil
}

func (m *MockEvidenceRepo) UpdateCorrectedTopicID(ctx context.Context, evidenceID, correctedTopicID int64) error {
	if m.Err != nil {
		return m.Err
	}
	if m.Evidences[evidenceID] != nil {
		m.Evidences[evidenceID].CorrectedTopicID = &correctedTopicID
	}
	return nil
}

type MockConceptRepo struct {
	Concepts map[string]*dal_model.TushareConcept
	Err      error
}

func NewMockConceptRepo() *MockConceptRepo {
	return &MockConceptRepo{
		Concepts: make(map[string]*dal_model.TushareConcept),
	}
}

func (m *MockConceptRepo) GetAll(ctx context.Context) ([]dal_model.TushareConcept, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var concepts []dal_model.TushareConcept
	for _, c := range m.Concepts {
		concepts = append(concepts, *c)
	}
	return concepts, nil
}

func (m *MockConceptRepo) GetByName(ctx context.Context, name string) (*dal_model.TushareConcept, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Concepts[name], nil
}

func (m *MockConceptRepo) Upsert(ctx context.Context, concept *dal_model.TushareConcept) error {
	if m.Err != nil {
		return m.Err
	}
	m.Concepts[concept.ConceptName] = concept
	return nil
}

type MockConceptDetailRepo struct {
	Details map[string][]dal_model.TushareConceptDetail
	Err     error
}

func NewMockConceptDetailRepo() *MockConceptDetailRepo {
	return &MockConceptDetailRepo{
		Details: make(map[string][]dal_model.TushareConceptDetail),
	}
}

func (m *MockConceptDetailRepo) GetByConceptName(ctx context.Context, conceptName string) ([]dal_model.TushareConceptDetail, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Details[conceptName], nil
}

func (m *MockConceptDetailRepo) GetByTsCode(ctx context.Context, tsCode string) ([]dal_model.TushareConceptDetail, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var result []dal_model.TushareConceptDetail
	for _, details := range m.Details {
		for _, d := range details {
			if d.TsCode == tsCode {
				result = append(result, d)
			}
		}
	}
	return result, nil
}

func (m *MockConceptDetailRepo) Upsert(ctx context.Context, detail *dal_model.TushareConceptDetail) error {
	if m.Err != nil {
		return m.Err
	}
	m.Details[detail.ConceptName] = append(m.Details[detail.ConceptName], *detail)
	return nil
}

type MockMarketSnapshotRepo struct {
	Snapshots map[string]*dal_model.MarketSnapshot
	Err       error
}

func NewMockMarketSnapshotRepo() *MockMarketSnapshotRepo {
	return &MockMarketSnapshotRepo{
		Snapshots: make(map[string]*dal_model.MarketSnapshot),
	}
}

func (m *MockMarketSnapshotRepo) GetByDate(ctx context.Context, date string) (*dal_model.MarketSnapshot, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Snapshots[date], nil
}

func (m *MockMarketSnapshotRepo) Upsert(ctx context.Context, snapshot *dal_model.MarketSnapshot) error {
	if m.Err != nil {
		return m.Err
	}
	m.Snapshots[snapshot.Date.Format("2006-01-02")] = snapshot
	return nil
}
