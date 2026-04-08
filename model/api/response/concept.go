// model/api/response/concept.go
package response

// ConceptItem 概念项
type ConceptItem struct {
	ConceptCode string `json:"concept_code"`
	ConceptName string `json:"concept_name"`
	Source      string `json:"source"`
}

// ConceptWithMapping 带映射信息的概念项
type ConceptWithMapping struct {
	ConceptCode string `json:"concept_code"`
	ConceptName string `json:"concept_name"`
	Source      string `json:"source"`
	IsMapped    bool   `json:"is_mapped"`
	TopicID     *int64 `json:"topic_id,omitempty"`
	TopicName   string `json:"topic_name,omitempty"`
}

// ConceptMappingItem 概念映射项
type ConceptMappingItem struct {
	ConceptCode string `json:"concept_code"`
	ConceptName string `json:"concept_name"`
	TopicID     int64  `json:"topic_id"`
	TopicName   string `json:"topic_name"`
	Source      string `json:"source"`
}

// ConceptListResp 概念列表响应
type ConceptListResp struct {
	Items []ConceptWithMapping `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
}
