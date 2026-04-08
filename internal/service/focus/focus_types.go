// internal/service/focus_types.go
package focus

import "time"

// FocusTopicResult 关注话题结果
type FocusTopicResult struct {
	ID              int64
	Name            string
	Category        string
	Source          string
	OccurrenceCount int
	FirstSeenDate   *time.Time
	LastSeenDate    *time.Time
}
