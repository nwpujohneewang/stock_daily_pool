// model/api/response/focus.go
package response

import "time"

// FocusTopicItem 关注话题项
type FocusTopicItem struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Category        string     `json:"category"`
	Source          string     `json:"source"`
	OccurrenceCount int        `json:"occurrence_count"`
	FirstSeenDate   *time.Time `json:"first_seen_date,omitempty"`
	LastSeenDate    *time.Time `json:"last_seen_date,omitempty"`
}
