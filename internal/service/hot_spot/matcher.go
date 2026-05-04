package hot_spot

import (
	"context"
	"encoding/json"
	"fmt"
	"stock/model/core_model"
	"strings"

	"go.uber.org/zap"
)

// StockProfile 股票画像
type StockProfile struct {
	TsCode       string   `json:"ts_code"`
	Name         string   `json:"name"`
	Industry     string   `json:"industry"`
	BusinessDesc string   `json:"business_desc"`
	Concepts     []string `json:"concepts"`
	HistTags     []string `json:"hist_tags"`
}

// Matcher 两路匹配：关键词召回 → LLM精筛
type Matcher struct {
	llm           LLMClient
	logger        *zap.Logger
	stockProfiles map[string]*StockProfile // tsCode → profile
	invertedIndex map[string][]string      // keyword → []tsCode
}

func NewMatcher(llm LLMClient, profiles []StockProfile, logger *zap.Logger) *Matcher {
	m := &Matcher{
		llm:           llm,
		logger:        logger,
		stockProfiles: make(map[string]*StockProfile, len(profiles)),
		invertedIndex: make(map[string][]string),
	}
	// 构建倒排索引
	for i := range profiles {
		p := &profiles[i]
		m.stockProfiles[p.TsCode] = p
		allKW := make(map[string]bool)
		allKW[p.Industry] = true
		allKW[p.Name] = true
		for _, c := range p.Concepts {
			allKW[c] = true
		}
		for _, t := range p.HistTags {
			allKW[t] = true
		}
		for kw := range allKW {
			if kw != "" {
				m.invertedIndex[kw] = append(m.invertedIndex[kw], p.TsCode)
			}
		}
	}
	logger.Info("matcher initialized",
		zap.Int("stocks", len(m.stockProfiles)),
		zap.Int("index_keywords", len(m.invertedIndex)))
	return m
}

// MatchStocks 关键词召回 → LLM精筛
func (m *Matcher) MatchStocks(ctx context.Context, topic core_model.HotTopic) ([]core_model.TopicStockRelation, error) {
	// 路径A：关键词召回
	candidates := m.keywordMatch(topic)
	if len(candidates) == 0 {
		return nil, nil
	}
	if len(candidates) > 100 {
		candidates = candidates[:100]
	}

	// 路径B：LLM精筛
	relations, err := m.llmFilter(ctx, topic, candidates)
	if err != nil {
		m.logger.Warn("llm filter failed, fallback", zap.Error(err))
		return m.fallbackRelations(topic, candidates), nil
	}
	return relations, nil
}

// keywordMatch 精确+包含匹配，按得分排序
func (m *Matcher) keywordMatch(topic core_model.HotTopic) []string {
	scoreMap := make(map[string]int)
	for _, keyword := range topic.Keywords {
		if codes, ok := m.invertedIndex[keyword]; ok {
			for _, code := range codes {
				scoreMap[code] += 3
			}
		}
		for idxKey, codes := range m.invertedIndex {
			if strings.Contains(idxKey, keyword) || strings.Contains(keyword, idxKey) {
				for _, code := range codes {
					scoreMap[code]++
				}
			}
		}
	}
	type scored struct {
		code  string
		score int
	}
	var candidates []scored
	for code, score := range scoreMap {
		if score >= 2 {
			candidates = append(candidates, scored{code, score})
		}
	}
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[i].score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	result := make([]string, 0, len(candidates))
	for _, c := range candidates {
		result = append(result, c.code)
	}
	return result
}

// llmFilter LLM精筛（每批30只股票）
func (m *Matcher) llmFilter(ctx context.Context, topic core_model.HotTopic, codes []string) ([]core_model.TopicStockRelation, error) {
	const batchSize = 30
	var all []core_model.TopicStockRelation

	for i := 0; i < len(codes); i += batchSize {
		end := i + batchSize
		if end > len(codes) {
			end = len(codes)
		}
		batch := codes[i:end]

		var rows []string
		for _, code := range batch {
			p := m.stockProfiles[code]
			if p == nil {
				continue
			}
			biz := p.BusinessDesc
			if len(biz) > 80 {
				biz = biz[:80] + "..."
			}
			rows = append(rows, fmt.Sprintf("| %s | %s | %s | %s |",
				p.TsCode, p.Name, p.Industry, biz))
		}

		userPrompt := fmt.Sprintf(`## 热点主题
名称：%s   摘要：%s   驱动：%s

## 候选股票
| 代码 | 名称 | 行业 | 主营业务 |
|------|------|------|---------|
%s

## 输出JSON数组
[{"code":"代码","relevance":"core/related/weak/irrelevant","reason":"10字原因"}]
只输出core和related的。`,
			topic.TopicName, topic.Summary, topic.Catalyst,
			strings.Join(rows, "\n"))

		resp, err := m.llm.ChatCompletion(ctx, "你是A股行业分析师。判断股票关联度。严格JSON。", userPrompt)
		if err != nil {
			return nil, err
		}

		var filterResults []struct {
			Code      string `json:"code"`
			Relevance string `json:"relevance"`
			Reason    string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(cleanJSON(resp)), &filterResults); err != nil {
			continue
		}

		for _, r := range filterResults {
			if r.Relevance != "core" && r.Relevance != "related" {
				continue
			}
			name := ""
			if p := m.stockProfiles[r.Code]; p != nil {
				name = p.Name
			}
			all = append(all, core_model.TopicStockRelation{
				TopicName: topic.TopicName, TsCode: r.Code,
				StockName: name, Relevance: r.Relevance, Reason: r.Reason,
			})
		}
	}
	return all, nil
}

// fallbackRelations 降级：直接用关键词结果
func (m *Matcher) fallbackRelations(topic core_model.HotTopic, codes []string) []core_model.TopicStockRelation {
	limit := 30
	if len(codes) < limit {
		limit = len(codes)
	}
	result := make([]core_model.TopicStockRelation, 0, limit)
	for _, code := range codes[:limit] {
		name := ""
		if p := m.stockProfiles[code]; p != nil {
			name = p.Name
		}
		result = append(result, core_model.TopicStockRelation{
			TopicName: topic.TopicName, TsCode: code,
			StockName: name, Relevance: "related", Reason: "关键词匹配",
		})
	}
	return result
}
