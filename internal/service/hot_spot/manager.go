package hot_spot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"stock/external/news"
	"stock/model/core_model"
)

type CachedHotTopic struct {
	core_model.HotTopic
	RelatedStocks []core_model.TopicStockRelation `json:"related_stocks"`
	CreatedAt     time.Time                       `json:"created_at"`
	UpdatedAt     time.Time                       `json:"updated_at"`
	Status        string                          `json:"status"` // active/fading/dead
}

type Manager struct {
	extractor *Extractor
	matcher   *Matcher
	rdb       *redis.Client
	newsAgg   *news.Aggregator
	logger    *zap.Logger
}

func NewManager(ext *Extractor, mat *Matcher, rdb *redis.Client, agg *news.Aggregator, l *zap.Logger) *Manager {
	return &Manager{extractor: ext, matcher: mat, rdb: rdb, newsAgg: agg, logger: l}
}

// Redis Key设计
func topicListKey(date string) string         { return fmt.Sprintf("hotspot:topics:%s", date) }
func topicDetailKey(date, name string) string { return fmt.Sprintf("hotspot:detail:%s:%s", date, name) }
func topicStocksKey(date, name string) string { return fmt.Sprintf("hotspot:stocks:%s:%s", date, name) }

// RefreshPreMarket 盘前全量刷新（08:00调用）
func (m *Manager) RefreshPreMarket(ctx context.Context, date string, existingTopics []string) error {
	start := time.Now()

	newsList, err := m.newsAgg.FetchAll(ctx, "")
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	if len(newsList) == 0 {
		return nil
	}

	topics, err := m.extractor.ExtractHotTopicsFromNews(ctx, newsList, existingTopics)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	for _, topic := range topics {
		relations, err := m.matcher.MatchStocks(ctx, topic)
		if err != nil {
			m.logger.Warn("match failed", zap.String("topic", topic.TopicName), zap.Error(err))
			continue
		}
		cached := CachedHotTopic{
			HotTopic: topic, RelatedStocks: relations,
			CreatedAt: time.Now(), UpdatedAt: time.Now(), Status: "active",
		}
		m.cacheHotTopic(ctx, date, cached)
	}

	m.logger.Info("pre-market refresh done",
		zap.Duration("elapsed", time.Since(start)),
		zap.Int("news", len(newsList)), zap.Int("topics", len(topics)))
	return nil
}

// RefreshIntraday 盘中增量刷新（每15min）
func (m *Manager) RefreshIntraday(ctx context.Context, date string, existingTopics []string) error {
	newsList, err := m.newsAgg.FetchHighImportance(ctx)
	if err != nil {
		return err
	}
	if len(newsList) == 0 {
		return nil
	}

	topics, err := m.extractor.ExtractHotTopicsFromNews(ctx, newsList, existingTopics)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		// 已存在 → 合并；新热点 → 匹配股票后缓存
		existing, _ := m.GetHotTopic(ctx, date, topic.TopicName)
		if existing != nil {
			existing.SourceNewsIDs = mergeSlice(existing.SourceNewsIDs, topic.SourceNewsIDs)
			existing.UpdatedAt = time.Now()
			m.cacheHotTopic(ctx, date, *existing)
			continue
		}
		relations, _ := m.matcher.MatchStocks(ctx, topic)
		cached := CachedHotTopic{
			HotTopic: topic, RelatedStocks: relations,
			CreatedAt: time.Now(), UpdatedAt: time.Now(), Status: "active",
		}
		m.cacheHotTopic(ctx, date, cached)
	}
	return nil
}

// GetAllHotTopics 供 classify/monitor 消费
func (m *Manager) GetAllHotTopics(ctx context.Context, date string) ([]CachedHotTopic, error) {
	members, err := m.rdb.SMembers(ctx, topicListKey(date)).Result()
	if err != nil {
		return nil, err
	}

	var topics []CachedHotTopic
	for _, name := range members {
		t, err := m.GetHotTopic(ctx, date, name)
		if err != nil || t == nil || t.Status != "active" {
			continue
		}
		topics = append(topics, *t)
	}
	return topics, nil
}

// GetHotTopic 获取单个热点
func (m *Manager) GetHotTopic(ctx context.Context, date, name string) (*CachedHotTopic, error) {
	data, err := m.rdb.Get(ctx, topicDetailKey(date, name)).Bytes()
	if err != nil {
		return nil, err
	}
	var t CachedHotTopic
	json.Unmarshal(data, &t)
	return &t, nil
}

func (m *Manager) cacheHotTopic(ctx context.Context, date string, t CachedHotTopic) {
	ttl := 24 * time.Hour
	detail, _ := json.Marshal(t)
	stocks, _ := json.Marshal(t.RelatedStocks)

	m.rdb.Set(ctx, topicDetailKey(date, t.TopicName), detail, ttl)
	m.rdb.Set(ctx, topicStocksKey(date, t.TopicName), stocks, ttl)
	m.rdb.SAdd(ctx, topicListKey(date), t.TopicName)
	m.rdb.Expire(ctx, topicListKey(date), ttl)
}

func mergeSlice(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	for _, s := range a {
		seen[s] = true
	}
	result := append([]string{}, a...)
	for _, s := range b {
		if !seen[s] {
			result = append(result, s)
		}
	}
	return result
}
