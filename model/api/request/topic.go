// model/api/request/topic.go
package request

// ListTopicReq 话题列表请求
type ListTopicReq struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=50"`
}

// Validate 校验并设置默认值
func (r *ListTopicReq) Validate() {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.PageSize <= 0 {
		r.PageSize = 50
	}
}
