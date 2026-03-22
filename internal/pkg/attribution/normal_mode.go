package attribution

import (
	"context"
	"time"
)

// NormalStrategy 普通模式：四维加权评分
type NormalStrategy struct{}

func (s *NormalStrategy) Name() string { return "normal" }

func (s *NormalStrategy) RunAttribution(ctx context.Context, input AttributionInput) (AttributionOutput, error) {
	allTopicActivity, err := getTopicActivityMap(ctx, input.Date)
	if err != nil {
		return AttributionOutput{}, err
	}

	maxLimitCount := 0
	for _, c := range allTopicActivity {
		if c > maxLimitCount {
			maxLimitCount = c
		}
	}

	totalHitCount := 0
	for _, m := range input.TopicRelations {
		totalHitCount += m.HitCount
	}

	var scores []TopicScore
	today, _ := time.Parse("2006-01-02", input.Date)

	for _, m := range input.TopicRelations {
		topicLimitCount, ok := allTopicActivity[int(m.TopicID)]
		if !ok || topicLimitCount <= 0 {
			continue
		}

		s1 := CalcActivityScore(topicLimitCount, maxLimitCount)

		s2 := 0.0
		if totalHitCount > 0 {
			s2 = CalcBindStrengthScore(m.HitCount, totalHitCount)
		}

		topicLimitTimes, _ := getTopicLimitTimes(ctx, input.Date, m.TopicID)
		s3 := CalcTimeProximityScore(input.QuoteTime, topicLimitTimes)

		var lastSeen time.Time
		if m.LastSeenDate != "" {
			if t, err := time.Parse("2006-01-02", m.LastSeenDate); err == nil {
				lastSeen = t
			}
		}
		s4 := CalcRecencyScore(lastSeen, today)

		total := WeightsNormal.WActivity*s1 +
			WeightsNormal.WBindStrength*s2 +
			WeightsNormal.WTimeProximity*s3 +
			WeightsNormal.WRecency*s4

		scores = append(scores, TopicScore{
			TopicID:       m.TopicID,
			TopicName:     m.TopicName,
			ActivityScore: s1,
			BindStrength:  s2,
			TimeProximity: s3,
			RecencyScore:  s4,
			TotalScore:    total,
			Source:        m.Source,
		})
	}

	return decideAttribution(scores, true), nil
}
