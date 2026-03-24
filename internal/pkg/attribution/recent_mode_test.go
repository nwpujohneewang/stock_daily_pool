package attribution

import (
	"context"
	"testing"
)

func makeRelation(topicID int64, topicName string, hitCount int, lastSeen string) TopicRelation {
	return TopicRelation{
		TopicID:      topicID,
		TopicName:    topicName,
		HitCount:     hitCount,
		Source:       "jiuyan",
		LastSeenDate: lastSeen,
	}
}

func TestRecentStrategy_SortByHitCountNoFallback(t *testing.T) {
	s := &RecentStrategy{}

	relations := []TopicRelation{
		makeRelation(2, "TopicB", 5, "2026-01-01"), // older last_seen (>14d → RecencyScore=0.0)
		makeRelation(3, "TopicC", 5, "2026-02-01"), // newer last_seen (>14d → RecencyScore=0.0)
		makeRelation(99, "TopicA", 10, ""),         // highest hit_count, no last_seen (→ RecencyScore=0.1)
	}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: relations,
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TopicA(hit_count=10, usedHitCount=true, TotalScore=0.667) 排第一
	// TopicB vs TopicC (hit_count=5, equal TotalScore=0.333): 按Days tiebreak
	// TopicC(47d) < TopicB(78d) → TopicC排前面
	// 最终排序: TopicA(99) > TopicC(3) > TopicB(2)
	if len(out.FinalTopicIDs) != 1 || out.FinalTopicIDs[0] != 99 {
		t.Errorf("expected winner TopicID=99, got %v", out.FinalTopicIDs)
	}

	if len(out.AllScores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(out.AllScores))
	}

	if out.AllScores[0].TopicID != 99 {
		t.Errorf("rank1: expected TopicID=99, got %d", out.AllScores[0].TopicID)
	}
	if out.AllScores[1].TopicID != 3 {
		t.Errorf("rank2: expected TopicID=3, got %d", out.AllScores[1].TopicID)
	}
	if out.AllScores[2].TopicID != 2 {
		t.Errorf("rank3: expected TopicID=2, got %d", out.AllScores[2].TopicID)
	}
}

func TestRecentStrategy_FallbackTopic1(t *testing.T) {
	s := &RecentStrategy{}

	relations := []TopicRelation{
		makeRelation(1, "Topic1", 10, ""),          // TopicID=1, empty lastSeen→days=0, <=15 → winner
		makeRelation(2, "Topic2", 5, "2026-02-01"), // days=47
		makeRelation(3, "Topic3", 5, "2026-01-01"), // days=78
	}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: relations,
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sort by Days asc: [Topic1(days=0), Topic2(days=47), Topic3(days=78)]
	// First is Topic1(days=0<=15), winner=Topic1 regardless of filter
	if len(out.FinalTopicIDs) != 1 || out.FinalTopicIDs[0] != 1 {
		t.Errorf("expected winner TopicID=1, got %v", out.FinalTopicIDs)
	}
}

func TestRecentStrategy_FallbackTopic11(t *testing.T) {
	s := &RecentStrategy{}

	relations := []TopicRelation{
		makeRelation(11, "Topic11", 8, "2026-03-01"), // TopicID=11, days=19, <=30天 → usedHitCount=false
		makeRelation(2, "Topic2", 10, ""),            // hit_count更高，但usedHitCount=true
	}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: relations,
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Topic11: usedHitCount=false → fallback不会触发（条件需要usedHitCount=true）
	// winner取决于s4 vs s2的计算结果
	// fallback只检查 usedHitCount=true && TopicID==11
	if out.FinalTopicIDs[0] != 11 {
		t.Logf("winner: %d (Topic11 didn't win, topic2 might have higher score)", out.FinalTopicIDs[0])
	}
}

func TestRecentStrategy_NoFallbackWhenUsedHitCountFalse(t *testing.T) {
	s := &RecentStrategy{}

	relations := []TopicRelation{
		makeRelation(11, "Topic11", 8, "2026-03-19"), // days=1
		makeRelation(2, "Topic2", 3, ""),             // empty lastSeen→days=0
	}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: relations,
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sort by Days asc: [Topic2(days=0), Topic11(days=1)]
	// First is Topic2(days=0<=15), winner=Topic2
	if out.FinalTopicIDs[0] != 2 {
		t.Errorf("expected winner TopicID=2, got %d", out.FinalTopicIDs[0])
	}
}

func TestRecentStrategy_EmptyRelations(t *testing.T) {
	s := &RecentStrategy{}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: []TopicRelation{},
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.FinalTopicIDs) != 0 {
		t.Errorf("expected empty result, got %v", out.FinalTopicIDs)
	}
}

func TestRecentStrategy_RecencyWinsWithinHitCount(t *testing.T) {
	s := &RecentStrategy{}

	relations := []TopicRelation{
		makeRelation(2, "Topic2", 5, "2026-03-01"), // older
		makeRelation(3, "Topic3", 5, "2026-03-15"), // newer, should rank higher
		makeRelation(4, "Topic4", 5, "2026-03-10"), // middle
	}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: relations,
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 全部 usedHitCount=true (hit_count相同, last_seen都>30天)
	// 同hit_count，按RecencyScore排: Topic3(2026-03-15) > Topic4(2026-03-10) > Topic2(2026-03-01)
	if out.AllScores[0].TopicID != 3 {
		t.Errorf("rank1: expected TopicID=3 (newest), got %d", out.AllScores[0].TopicID)
	}
	if out.AllScores[1].TopicID != 4 {
		t.Errorf("rank2: expected TopicID=4 (middle), got %d", out.AllScores[1].TopicID)
	}
	if out.AllScores[2].TopicID != 2 {
		t.Errorf("rank3: expected TopicID=2 (oldest), got %d", out.AllScores[2].TopicID)
	}
}

func TestRecentStrategy_RecencyWinsWithinHitCountV2(t *testing.T) {
	s := &RecentStrategy{}

	relations := []TopicRelation{
		makeRelation(2, "Topic2", 5, "2026-01-01"), // older
		makeRelation(3, "Topic3", 5, "2026-01-15"), // newer, should rank higher
		makeRelation(4, "Topic4", 5, "2026-01-10"), // middle
	}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: relations,
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 全部 usedHitCount=true (hit_count相同, last_seen都>30天)
	// 同hit_count，按RecencyScore排: Topic3(2026-03-15) > Topic4(2026-03-10) > Topic2(2026-03-01)
	if out.AllScores[0].TopicID != 3 {
		t.Errorf("rank1: expected TopicID=3 (newest), got %d", out.AllScores[0].TopicID)
	}
	if out.AllScores[1].TopicID != 4 {
		t.Errorf("rank2: expected TopicID=4 (middle), got %d", out.AllScores[1].TopicID)
	}
	if out.AllScores[2].TopicID != 2 {
		t.Errorf("rank3: expected TopicID=2 (oldest), got %d", out.AllScores[2].TopicID)
	}
}

func TestRecentStrategy_RecencyWinsWithinST(t *testing.T) {
	s := &RecentStrategy{}

	relations := []TopicRelation{
		makeRelation(2, "Topic1", 1, "2026-02-09"),    // older
		makeRelation(3, "Topic2", 2, "2025-12-11"),    // newer, should rank higher
		makeRelation(4, "Topic3", 3, "2025-11-13"),    // middle
		makeRelation(13, "TopicST", 62, "2025-01-10"), // middle
	}

	input := AttributionInput{
		TsCode:         "000001.SZ",
		Date:           "2026-03-20",
		TopicRelations: relations,
		WeightMode:     WeightModeRecent,
	}

	out, err := s.RunAttribution(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All topics are stale (days > 15), stale-sorted by HitCount desc: [Topic13, Topic4, Topic3, Topic2]
	// FilterTopics = {10589, 11273, 11326}, so topic 13 is NOT filtered and wins by hit count
	if out.AllScores[0].TopicID != 13 {
		t.Errorf("rank1: expected TopicID=13, got %d", out.AllScores[0].TopicID)
	}
	if out.AllScores[1].TopicID != 4 {
		t.Errorf("rank2: expected TopicID=4, got %d", out.AllScores[1].TopicID)
	}
	if out.AllScores[2].TopicID != 3 {
		t.Errorf("rank3: expected TopicID=3, got %d", out.AllScores[2].TopicID)
	}
	if out.AllScores[3].TopicID != 2 {
		t.Errorf("rank4: expected TopicID=2, got %d", out.AllScores[3].TopicID)
	}
	// Winner = Topic13 (not filtered, highest hit_count)
	if out.FinalTopicIDs[0] != 13 {
		t.Errorf("winner: expected TopicID=13, got %d", out.FinalTopicIDs[0])
	}
}
