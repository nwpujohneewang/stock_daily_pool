package mocks

import (
	"context"
	"fmt"
	"stock/model/dal_model"
)

type MockQuoteCache struct {
	Quotes map[string]dal_model.StockQuote
	Err    error
}

func NewMockQuoteCache() *MockQuoteCache {
	return &MockQuoteCache{
		Quotes: make(map[string]dal_model.StockQuote),
	}
}

func (m *MockQuoteCache) Set(ctx context.Context, quote *dal_model.StockQuote) error {
	if m.Err != nil {
		return m.Err
	}
	m.Quotes[quote.TsCode] = *quote
	return nil
}

func (m *MockQuoteCache) Get(ctx context.Context, tsCode string) (*dal_model.StockQuote, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	q, ok := m.Quotes[tsCode]
	if !ok {
		return nil, nil
	}
	return &q, nil
}

func (m *MockQuoteCache) Delete(ctx context.Context, tsCode string) error {
	if m.Err != nil {
		return m.Err
	}
	delete(m.Quotes, tsCode)
	return nil
}

type MockPoolCache struct {
	LimitUpMembers map[string][]string
	Above5Members  map[string][]string
	FirstLimit     map[string]map[string]string
	Err            error
}

func NewMockPoolCache() *MockPoolCache {
	return &MockPoolCache{
		LimitUpMembers: make(map[string][]string),
		Above5Members:  make(map[string][]string),
		FirstLimit:     make(map[string]map[string]string),
	}
}

func (m *MockPoolCache) AddLimitUp(ctx context.Context, date, tsCode string) error {
	if m.Err != nil {
		return m.Err
	}
	m.LimitUpMembers[date] = append(m.LimitUpMembers[date], tsCode)
	return nil
}

func (m *MockPoolCache) RemoveLimitUp(ctx context.Context, date, tsCode string) error {
	if m.Err != nil {
		return m.Err
	}
	pools := m.LimitUpMembers[date]
	for i, c := range pools {
		if c == tsCode {
			m.LimitUpMembers[date] = append(pools[:i], pools[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockPoolCache) GetLimitUpMembers(ctx context.Context, date string) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.LimitUpMembers[date], nil
}

func (m *MockPoolCache) AddAbove5(ctx context.Context, date, tsCode string) error {
	if m.Err != nil {
		return m.Err
	}
	m.Above5Members[date] = append(m.Above5Members[date], tsCode)
	return nil
}

func (m *MockPoolCache) RemoveAbove5(ctx context.Context, date, tsCode string) error {
	if m.Err != nil {
		return m.Err
	}
	pools := m.Above5Members[date]
	for i, c := range pools {
		if c == tsCode {
			m.Above5Members[date] = append(pools[:i], pools[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockPoolCache) GetAbove5Members(ctx context.Context, date string) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Above5Members[date], nil
}

func (m *MockPoolCache) SetFirstLimitTime(ctx context.Context, date, tsCode, limitTime string) error {
	if m.Err != nil {
		return m.Err
	}
	if m.FirstLimit[date] == nil {
		m.FirstLimit[date] = make(map[string]string)
	}
	m.FirstLimit[date][tsCode] = limitTime
	return nil
}

func (m *MockPoolCache) GetFirstLimitTime(ctx context.Context, date, tsCode string) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	if m.FirstLimit[date] == nil {
		return "", nil
	}
	return m.FirstLimit[date][tsCode], nil
}

type MockFocusCache struct {
	FocusTopics map[string]map[int64]bool
	Err         error
}

func NewMockFocusCache() *MockFocusCache {
	return &MockFocusCache{
		FocusTopics: make(map[string]map[int64]bool),
	}
}

func (m *MockFocusCache) SetFocusTopics(ctx context.Context, date string, topicIDs []int64) error {
	if m.Err != nil {
		return m.Err
	}
	if m.FocusTopics[date] == nil {
		m.FocusTopics[date] = make(map[int64]bool)
	}
	for _, id := range topicIDs {
		m.FocusTopics[date][id] = true
	}
	return nil
}

func (m *MockFocusCache) GetFocusTopics(ctx context.Context, date string) ([]int64, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var ids []int64
	if m.FocusTopics[date] != nil {
		for id := range m.FocusTopics[date] {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (m *MockFocusCache) IsFocused(ctx context.Context, date string, topicID int64) (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	if m.FocusTopics[date] == nil {
		return false, nil
	}
	return m.FocusTopics[date][topicID], nil
}

func (m *MockFocusCache) RemoveFocusTopic(ctx context.Context, date string, topicID int64) error {
	if m.Err != nil {
		return m.Err
	}
	if m.FocusTopics[date] != nil {
		delete(m.FocusTopics[date], topicID)
	}
	return nil
}

func (m *MockFocusCache) ClearFocusTopics(ctx context.Context, date string) error {
	if m.Err != nil {
		return m.Err
	}
	delete(m.FocusTopics, date)
	return nil
}

type MockMappingCache struct {
	StockTopics  map[string][]dal_model.TopicMapping
	BindStrength map[string]int
	Err          error
}

func NewMockMappingCache() *MockMappingCache {
	return &MockMappingCache{
		StockTopics:  make(map[string][]dal_model.TopicMapping),
		BindStrength: make(map[string]int),
	}
}

func (m *MockMappingCache) SetStockTopics(ctx context.Context, tsCode string, mappings []dal_model.TopicMapping) error {
	if m.Err != nil {
		return m.Err
	}
	m.StockTopics[tsCode] = mappings
	return nil
}

func (m *MockMappingCache) GetStockTopics(ctx context.Context, tsCode string) ([]dal_model.TopicMapping, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.StockTopics[tsCode], nil
}

func (m *MockMappingCache) GetBindStrength(ctx context.Context, tsCode string, topicID int64) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	key := tsCode + ":" + string(rune(topicID))
	return m.BindStrength[key], nil
}

func (m *MockMappingCache) SetBindStrength(ctx context.Context, tsCode string, topicID int64, count int) error {
	if m.Err != nil {
		return m.Err
	}
	key := tsCode + ":" + string(rune(topicID))
	m.BindStrength[key] = count
	return nil
}

func (m *MockMappingCache) GetAllBindStrength(ctx context.Context, tsCode string) (map[int64]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	result := make(map[int64]int)
	for key, val := range m.BindStrength {
		var sid string
		var tid int64
		fmt.Sscanf(key, "%s:%d", &sid, &tid)
		if sid == tsCode {
			result[tid] = val
		}
	}
	return result, nil
}

type MockConceptCache struct {
	StockConcepts map[string][]string
	ConceptTopics map[string][]int64
	Err           error
}

func NewMockConceptCache() *MockConceptCache {
	return &MockConceptCache{
		StockConcepts: make(map[string][]string),
		ConceptTopics: make(map[string][]int64),
	}
}

func (m *MockConceptCache) SetStockConcepts(ctx context.Context, tsCode string, concepts []string) error {
	if m.Err != nil {
		return m.Err
	}
	m.StockConcepts[tsCode] = concepts
	return nil
}

func (m *MockConceptCache) GetStockConcepts(ctx context.Context, tsCode string) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.StockConcepts[tsCode], nil
}

func (m *MockConceptCache) SetConceptTopics(ctx context.Context, conceptName string, topicIDs []int64) error {
	if m.Err != nil {
		return m.Err
	}
	m.ConceptTopics[conceptName] = topicIDs
	return nil
}

func (m *MockConceptCache) GetConceptTopics(ctx context.Context, conceptName string) ([]int64, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.ConceptTopics[conceptName], nil
}

func (m *MockConceptCache) GetAllStockConcepts(ctx context.Context) (map[string][]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.StockConcepts, nil
}

type MockActivityCache struct {
	TopicActivity map[string]map[int64]int
	Err           error
}

func NewMockActivityCache() *MockActivityCache {
	return &MockActivityCache{
		TopicActivity: make(map[string]map[int64]int),
	}
}

func (m *MockActivityCache) IncrTopicLimitCount(ctx context.Context, date string, topicID int64) error {
	if m.Err != nil {
		return m.Err
	}
	if m.TopicActivity[date] == nil {
		m.TopicActivity[date] = make(map[int64]int)
	}
	m.TopicActivity[date][topicID]++
	return nil
}

func (m *MockActivityCache) DecrTopicLimitCount(ctx context.Context, date string, topicID int64) error {
	if m.Err != nil {
		return m.Err
	}
	if m.TopicActivity[date] != nil && m.TopicActivity[date][topicID] > 0 {
		m.TopicActivity[date][topicID]--
	}
	return nil
}

func (m *MockActivityCache) GetTopicLimitCount(ctx context.Context, date string, topicID int64) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	if m.TopicActivity[date] == nil {
		return 0, nil
	}
	return m.TopicActivity[date][topicID], nil
}

func (m *MockActivityCache) GetAllTopicLimitCounts(ctx context.Context, date string) (map[int64]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.TopicActivity[date], nil
}

type MockAlertDedupCache struct {
	Alerted map[string]bool
	Err     error
}

func NewMockAlertDedupCache() *MockAlertDedupCache {
	return &MockAlertDedupCache{
		Alerted: make(map[string]bool),
	}
}

func (m *MockAlertDedupCache) IsAlerted(ctx context.Context, date, tsCode string, topicID int64) (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	key := date + ":" + tsCode + ":" + string(rune(topicID))
	return m.Alerted[key], nil
}

func (m *MockAlertDedupCache) SetAlerted(ctx context.Context, date, tsCode string, topicID int64) error {
	if m.Err != nil {
		return m.Err
	}
	key := date + ":" + tsCode + ":" + string(rune(topicID))
	m.Alerted[key] = true
	return nil
}

func (m *MockAlertDedupCache) GetAlertedTopics(ctx context.Context, date, tsCode string) ([]int64, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return []int64{}, nil
}

type MockSnapshotCache struct {
	Snapshots map[string]interface{}
	Err       error
}

func NewMockSnapshotCache() *MockSnapshotCache {
	return &MockSnapshotCache{
		Snapshots: make(map[string]interface{}),
	}
}

func (m *MockSnapshotCache) Set(ctx context.Context, key string, data interface{}) error {
	if m.Err != nil {
		return m.Err
	}
	m.Snapshots[key] = data
	return nil
}

func (m *MockSnapshotCache) Get(ctx context.Context, key string) (interface{}, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Snapshots[key], nil
}

func (m *MockSnapshotCache) Delete(ctx context.Context, key string) error {
	if m.Err != nil {
		return m.Err
	}
	delete(m.Snapshots, key)
	return nil
}

func (m *MockSnapshotCache) Exists(ctx context.Context, key string) (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	_, ok := m.Snapshots[key]
	return ok, nil
}
