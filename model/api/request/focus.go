// model/api/request/focus.go
package request

// SetFocusReq 设置关注话题请求
type SetFocusReq struct {
	Date     string  `json:"date"`
	TopicIDs []int64 `json:"topic_ids" binding:"required"`
}

// GetFocusReq 获取关注话题请求
type GetFocusReq struct {
	Date string `form:"date"`
}
