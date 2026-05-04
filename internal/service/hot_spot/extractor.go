package hot_spot

import (
	"context"
	"encoding/json"
	"fmt"
	"stock/model/core_model"
	"strings"
	"time"

	"go.uber.org/zap"
)

// LLMClient 复用已有 external/llm 包的接口
type LLMClient interface {
	ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type Extractor struct {
	llm    LLMClient
	logger *zap.Logger
}

func NewExtractor(llm LLMClient, logger *zap.Logger) *Extractor {
	return &Extractor{llm: llm, logger: logger}
}

// ━━━━━━━━━━ 第一层：批量新闻结构化提取 ━━━━━━━━━━

const extractSystemPrompt = `你是A股市场新闻分析师。对每条新闻提取结构化信息，严格输出JSON数组。`

func buildExtractUserPrompt(batch []core_model.RawNews) string {
	var sb strings.Builder
	sb.WriteString("请对以下每条新闻提取结构化信息。\n\n## 新闻列表\n")
	for _, n := range batch {
		content := n.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s] %s：%s\n", n.NewsID, n.Title, content))
	}
	sb.WriteString(`
## 输出格式（严格JSON数组）
[
  {
    "news_id": "原始ID",
    "summary": "一句话摘要（20字内）",
    "keywords": ["关键词1", "关键词2", "关键词3"],
    "sector": "涉及行业/概念（如'AI算力'而非'AI'）",
    "impact": "利好/利空/中性",
    "importance": "high/medium/low",
    "event_type": "政策/业绩/行业动态/技术突破/市场情绪"
  }
]
`)
	return sb.String()
}

// ExtractBatch 每批25条，减少LLM调用次数
func (e *Extractor) ExtractBatch(ctx context.Context, newsList []core_model.RawNews) ([]core_model.NewsExtractResult, error) {
	const batchSize = 25
	var allResults []core_model.NewsExtractResult

	for i := 0; i < len(newsList); i += batchSize {
		end := i + batchSize
		if end > len(newsList) {
			end = len(newsList)
		}
		batch := newsList[i:end]

		resp, err := e.llm.ChatCompletion(ctx, extractSystemPrompt, buildExtractUserPrompt(batch))
		if err != nil {
			e.logger.Warn("extract batch failed", zap.Int("start", i), zap.Error(err))
			continue
		}

		var results []core_model.NewsExtractResult
		if err := json.Unmarshal([]byte(cleanJSON(resp)), &results); err != nil {
			e.logger.Warn("parse failed", zap.Error(err))
			continue
		}
		allResults = append(allResults, results...)
	}
	return allResults, nil
}

// ━━━━━━━━━━ 第二层：关键词聚类（纯算法，无LLM开销） ━━━━━━━━━━

type NewsCluster struct {
	ID            int
	PrimarySector string
	NewsItems     []core_model.NewsExtractResult
	Keywords      map[string]int
}

// ClusterBySector 按sector粗分 + keywords Jaccard相似度细分
func ClusterBySector(extracted []core_model.NewsExtractResult) []NewsCluster {
	// Step 1: 按sector粗分组
	sectorGroups := make(map[string][]core_model.NewsExtractResult)
	for _, n := range extracted {
		if n.Importance == "low" {
			continue
		}
		sector := normalizeSector(n.Sector)
		sectorGroups[sector] = append(sectorGroups[sector], n)
	}

	var clusters []NewsCluster
	clusterID := 0

	for sector, group := range sectorGroups {
		if len(group) < 1 {
			continue
		}

		// 小组直接成簇
		if len(group) <= 3 {
			clusters = append(clusters, NewsCluster{
				ID: clusterID, PrimarySector: sector,
				NewsItems: group, Keywords: mergeKeywords(group),
			})
			clusterID++
			continue
		}

		// 大组按keywords重叠度细分
		subClusters := splitByKeywordOverlap(group, 0.3)
		for _, sub := range subClusters {
			clusters = append(clusters, NewsCluster{
				ID: clusterID, PrimarySector: sector,
				NewsItems: sub, Keywords: mergeKeywords(sub),
			})
			clusterID++
		}
	}
	return clusters
}

func splitByKeywordOverlap(items []core_model.NewsExtractResult, threshold float64) [][]core_model.NewsExtractResult {
	assigned := make([]bool, len(items))
	var clusters [][]core_model.NewsExtractResult

	for i := 0; i < len(items); i++ {
		if assigned[i] {
			continue
		}
		cluster := []core_model.NewsExtractResult{items[i]}
		assigned[i] = true

		for j := i + 1; j < len(items); j++ {
			if assigned[j] {
				continue
			}
			if keywordOverlap(items[i].Keywords, items[j].Keywords) >= threshold {
				cluster = append(cluster, items[j])
				assigned[j] = true
			}
		}
		clusters = append(clusters, cluster)
	}
	return clusters
}

// keywordOverlap Jaccard相似度
func keywordOverlap(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setA := make(map[string]bool, len(a))
	for _, k := range a {
		setA[k] = true
	}
	intersection := 0
	for _, k := range b {
		if setA[k] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func mergeKeywords(items []core_model.NewsExtractResult) map[string]int {
	kw := make(map[string]int)
	for _, item := range items {
		for _, k := range item.Keywords {
			kw[k]++
		}
	}
	return kw
}

func normalizeSector(sector string) string {
	sector = strings.TrimSpace(sector)
	sector = strings.TrimSuffix(sector, "概念")
	sector = strings.TrimSuffix(sector, "板块")
	return sector
}

// ━━━━━━━━━━ 第三层：LLM生成热点主题卡片 ━━━━━━━━━━

const hotspotSystemPrompt = `你是A股热点分析师。基于新闻摘要组生成热点主题卡片。严格输出JSON。`

func buildHotspotUserPrompt(cluster NewsCluster, existingTopics []string) string {
	var sb strings.Builder
	sb.WriteString("## 新闻摘要组\n")
	for _, n := range cluster.NewsItems {
		sb.WriteString(fmt.Sprintf("- [%s] %s（关键词：%s）\n",
			n.Importance, n.Summary, strings.Join(n.Keywords, ",")))
	}
	sb.WriteString("\n## 已有标签（优先复用）\n")
	if len(existingTopics) > 0 {
		sb.WriteString(strings.Join(existingTopics, "、"))
	} else {
		sb.WriteString("（暂无）")
	}
	sb.WriteString(`

## 输出格式（严格JSON）
{
  "topic_name": "热点名称（4-8字，如'AI算力'）",
  "topic_l2": "细分概念标签",
  "summary": "50字内热点摘要",
  "catalyst": "驱动因素",
  "keywords": ["匹配关键词5-10个，含行业术语/产品名/技术名"],
  "hot_level": "high/medium/low",
  "related_stock_hints": ["受益公司特征，如'有GPU服务器业务的公司'"]
}

## 规则
1. topic_name精准（"AI"不行，"AI算力"可以）
2. 优先复用已有标签名
`)
	return sb.String()
}

// GenerateHotTopics 从每个新闻簇生成热点主题
func (e *Extractor) GenerateHotTopics(ctx context.Context, clusters []NewsCluster, existingTopics []string) ([]core_model.HotTopic, error) {
	var topics []core_model.HotTopic

	for _, cluster := range clusters {
		hasImportant := false
		for _, n := range cluster.NewsItems {
			if n.Importance == "high" || n.Importance == "medium" {
				hasImportant = true
				break
			}
		}
		if !hasImportant {
			continue
		}

		resp, err := e.llm.ChatCompletion(ctx, hotspotSystemPrompt, buildHotspotUserPrompt(cluster, existingTopics))
		if err != nil {
			e.logger.Warn("generate hotspot failed", zap.Error(err))
			continue
		}

		var topic core_model.HotTopic
		if err := json.Unmarshal([]byte(cleanJSON(resp)), &topic); err != nil {
			e.logger.Warn("parse failed", zap.Error(err))
			continue
		}

		for _, n := range cluster.NewsItems {
			topic.SourceNewsIDs = append(topic.SourceNewsIDs, n.NewsID)
		}
		topics = append(topics, topic)
	}
	return topics, nil
}

// ━━━━━━━━━━ 完整Pipeline ━━━━━━━━━━

// ExtractHotTopicsFromNews 一键：新闻→结构化→聚类→热点
func (e *Extractor) ExtractHotTopicsFromNews(
	ctx context.Context,
	newsList []core_model.RawNews,
	existingTopics []string,
) ([]core_model.HotTopic, error) {
	start := time.Now()

	// 第一层：结构化提取
	extracted, err := e.ExtractBatch(ctx, newsList)
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}
	if len(extracted) == 0 {
		return nil, nil
	}

	// 第二层：关键词聚类（零LLM开销）
	clusters := ClusterBySector(extracted)

	// 第三层：LLM生成热点
	topics, err := e.GenerateHotTopics(ctx, clusters, existingTopics)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	e.logger.Info("pipeline done",
		zap.Duration("elapsed", time.Since(start)),
		zap.Int("news", len(newsList)),
		zap.Int("clusters", len(clusters)),
		zap.Int("topics", len(topics)))
	return topics, nil
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
