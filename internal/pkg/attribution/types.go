package attribution

import (
	"context"
	"slices"
	"sort"
	dalmodel "stock/model/dal_model"
	"time"
)

type TopicRelation = dalmodel.TopicRelation

// AttributionInput 归因算法输入
type AttributionInput struct {
	TsCode         string          `json:"ts_code"`
	StockName      string          `json:"stock_name"`
	QuoteTime      time.Time       `json:"quote_time"`      // 该股票涨停/入池时间
	Date           string          `json:"date"`            // 交易日 YYYY-MM-DD
	TopicRelations []TopicRelation `json:"topic_relations"` // 候选热点映射列表
	WeightMode     WeightMode      `json:"weight_mode"`     // 权重模式
}

// WeightMode 权重模式
type WeightMode int

const (
	WeightModeNormal WeightMode = 0 // 普通模式（有韭研标签）
	WeightModeRecent WeightMode = 1 // 最近模式（last_seen > 30d → 只看 hit_count; 否则只看 last_seen）
)

var FilterTopics = []int64{
	10589,
	11273,
	11326,
	10884,
	11081,
}

// AttributionStrategy 归因策略接口
type AttributionStrategy interface {
	Name() string
	RunAttribution(ctx context.Context, input AttributionInput) (AttributionOutput, error)
}

// NewStrategy 根据 WeightMode 创建对应的策略实例
func NewStrategy(mode WeightMode) AttributionStrategy {
	switch mode {
	case WeightModeRecent:
		return &RecentStrategy{}
	default:
		return &NormalStrategy{}
	}
}

// AttributionOutput 归因算法输出
type AttributionOutput struct {
	FinalTopicIDs     []int64      `json:"final_topic_ids"`     // 归因结果（1~2个）
	AllScores         []TopicScore `json:"all_scores"`          // 所有候选评分（降序）
	IsDualAttribution bool         `json:"is_dual_attribution"` // 是否双归因
	Confidence        float64      `json:"confidence"`          // 最高得分作为置信度
}

// TopicScore 是单个热点的归因分数（四维分数）
type TopicScore struct {
	TopicID       int64   `json:"topic_id"`
	TopicName     string  `json:"topic_name"`
	Category      string  `json:"category"`
	ActivityScore float64 `json:"s_activity"`
	BindStrength  float64 `json:"s_bind_strength"`
	TimeProximity float64 `json:"s_time_proximity"`
	RecencyScore  float64 `json:"s_recency"`
	TotalScore    float64 `json:"total_score"`
	Source        string  `json:"source"`
	UsedHitCount  bool    `json:"used_hit_count"` // true = TotalScore was set from hit_count (last_seen > 30d)
	Days          int     `json:"days"`           // 原始天数差，用于同分时tiebreak
}

// AttributionWeights 四维权重配置
type AttributionWeights struct {
	WActivity      float64 `json:"w_activity"`       // 活跃度权重
	WBindStrength  float64 `json:"w_bind_strength"`  // 关联强度权重
	WTimeProximity float64 `json:"w_time_proximity"` // 时间接近度权重
	WRecency       float64 `json:"w_recency"`        // 时效性权重
}

// 权重配置（对外暴露，便于测试/使用）
var WeightsNormal = AttributionWeights{
	WActivity:      0.15,
	WBindStrength:  0.40,
	WTimeProximity: 0.20,
	WRecency:       0.25,
}

// ---------------------- decideAttribution ----------------------

func decideAttribution(scores []TopicScore, allowDual bool) AttributionOutput {
	var out AttributionOutput
	if len(scores) == 0 {
		return out
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	if len(scores) == 1 {
		out.FinalTopicIDs = []int64{scores[0].TopicID}
		out.Confidence = scores[0].TotalScore
		out.IsDualAttribution = false
		out.AllScores = scores
		return out
	}

	top1 := scores[0]
	top2 := scores[1]

	if allowDual && top1.TotalScore-top2.TotalScore < 0.1 {
		out.FinalTopicIDs = []int64{top1.TopicID, top2.TopicID}
		out.IsDualAttribution = true
		out.Confidence = top1.TotalScore
	} else {
		out.FinalTopicIDs = []int64{top1.TopicID}
		out.IsDualAttribution = false
		out.Confidence = top1.TotalScore
	}
	out.AllScores = scores
	return out
}

func InFilterTopic(topicID int64) bool {
	return slices.Contains(FilterTopics, topicID)
}
