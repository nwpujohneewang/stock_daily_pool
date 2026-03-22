package attribution

import (
	"context"
	"sort"
	"time"
)

type RecentStrategy struct{}

func (s *RecentStrategy) Name() string { return "recent" }

func (s *RecentStrategy) RunAttribution(ctx context.Context, input AttributionInput) (AttributionOutput, error) {
	today, _ := time.Parse("2006-01-02", input.Date)

	var scores []TopicScore
	for _, m := range input.TopicRelations {
		days := 0
		if m.LastSeenDate != "" {
			if t, err := time.Parse("2006-01-02", m.LastSeenDate); err == nil {
				days = int(today.Sub(t).Hours() / 24)
			}
		}
		scores = append(scores, TopicScore{
			TopicID:      m.TopicID,
			TopicName:    m.TopicName,
			BindStrength: float64(m.HitCount),
			RecencyScore: 0,
			TotalScore:   0,
			Source:       m.Source,
			UsedHitCount: false,
			Days:         days,
		})
	}

	if len(scores) == 0 {
		return AttributionOutput{}, nil
	}

	const staleDays = 15

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Days < scores[j].Days
	})

	winner := int64(0)

	if scores[0].Days <= staleDays {
		winner = scores[0].TopicID
	} else {
		var stale []TopicScore
		for i := range scores {
			if scores[i].Days > staleDays {
				stale = append(stale, scores[i])
			}
		}
		sort.Slice(stale, func(i, j int) bool {
			if stale[i].BindStrength != stale[j].BindStrength {
				return stale[i].BindStrength > stale[j].BindStrength
			}
			return stale[i].Days < stale[j].Days
		})
		for i := range stale {
			if !InFilterTopic(stale[i].TopicID) {
				winner = stale[i].TopicID
				break
			}
		}
		scores = stale
	}

	return AttributionOutput{
		FinalTopicIDs:     []int64{winner},
		AllScores:         scores,
		IsDualAttribution: false,
		Confidence:        1.0,
	}, nil
}
