// model/api/request/topic_dict.go
package request

// CreateTopicDictReq 创建话题词典请求
type CreateTopicDictReq struct {
	RawTopicName   string `json:"raw_topic_name" binding:"required"`
	NormalizedName string `json:"normalized_name" binding:"required"`
	Category       string `json:"category"`
}

// UpdateTopicDictReq 更新话题词典请求
type UpdateTopicDictReq struct {
	RawTopicName   string `json:"raw_topic_name"`
	NormalizedName string `json:"normalized_name"`
	Category       string `json:"category"`
}
