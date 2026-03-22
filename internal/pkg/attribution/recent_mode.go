package attribution

import (
	"context"
	"sort"
	"time"
)

// RecentStrategy 最近模式：条件选择 S2 或 S4
// - last_seen 为空或 > 30天 → 只看 hit_count (S2)
// - last_seen <= 30天 → 只看 last_seen (S4)
type RecentStrategy struct{}

func (s *RecentStrategy) Name() string { return "recent" }

func (s *RecentStrategy) RunAttribution(ctx context.Context, input AttributionInput) (AttributionOutput, error) {
	totalHitCount := 0
	for _, m := range input.TopicRelations {
		totalHitCount += m.HitCount
	}

	var scores []TopicScore
	today, _ := time.Parse("2006-01-02", input.Date)

	for _, m := range input.TopicRelations {
		s2 := 0.0
		if totalHitCount > 0 {
			s2 = CalcBindStrengthScore(m.HitCount, totalHitCount)
		}

		var lastSeen time.Time
		days := 0
		if m.LastSeenDate != "" {
			if t, err := time.Parse("2006-01-02", m.LastSeenDate); err == nil {
				lastSeen = t
				days = int(today.Sub(lastSeen).Hours() / 24)
			}
		}
		s4 := CalcRecencyScore(lastSeen, today)

		var total float64
		usedHitCount := false

		if !lastSeen.IsZero() {
			if days >= 0 && days <= 30 {
				total = s4
			} else {
				total = s2
				usedHitCount = true
			}
		} else {
			total = s2
			usedHitCount = true
		}

		scores = append(scores, TopicScore{
			TopicID:       m.TopicID,
			TopicName:     m.TopicName,
			ActivityScore: 0,
			BindStrength:  s2,
			TimeProximity: 0,
			RecencyScore:  s4,
			TotalScore:    total,
			Source:        m.Source,
			UsedHitCount:  usedHitCount,
			Days:          days,
		})
	}

	if len(scores) == 0 {
		return AttributionOutput{}, nil
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].TotalScore != scores[j].TotalScore {
			return scores[i].TotalScore > scores[j].TotalScore
		}
		return scores[i].Days < scores[j].Days
	})

	top1 := scores[0]

	if top1.UsedHitCount && (top1.TopicID == 1 || top1.TopicID == 11) {
		bestIdx := -1
		for i := 1; i < len(scores); i++ {
			if scores[i].TopicID != 1 && scores[i].TopicID != 11 {
				if bestIdx == -1 || scores[i].Days < scores[bestIdx].Days {
					bestIdx = i
				}
			}
		}
		if bestIdx > 0 {
			scores[0], scores[bestIdx] = scores[bestIdx], scores[0]
		}
	}

	return AttributionOutput{
		FinalTopicIDs:     []int64{scores[0].TopicID},
		AllScores:         scores,
		IsDualAttribution: false,
		Confidence:        scores[0].TotalScore,
	}, nil
}
