// model/api/request/concept.go
package request

// ListConceptReq 概念列表请求
type ListConceptReq struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=50"`
}

// Validate 校验并设置默认值
func (r *ListConceptReq) Validate() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 200 {
		r.PageSize = 50
	}
}

// CreateConceptMappingReq 创建概念映射请求
type CreateConceptMappingReq struct {
	ConceptName string `json:"concept_name" binding:"required"`
	ConceptCode string `json:"concept_code" binding:"required"`
	TopicID     int64  `json:"topic_id" binding:"required"`
}
