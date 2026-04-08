// internal/service/topic_dict_types.go
package topic

import "errors"

// TopicDictParams 创建/更新话题词典参数
type TopicDictParams struct {
	RawTopicName   string
	NormalizedName string
	Category       string
}

// TopicDictResult 话题词典结果
type TopicDictResult struct {
	ID             int64
	RawTopicName   string
	NormalizedName string
	Category       string
}

// ErrTopicDictNotFound 词典项不存在
var ErrTopicDictNotFound = errors.New("topic dictionary not found")

// ErrTopicDictDuplicate 词典项重复
var ErrTopicDictDuplicate = errors.New("topic dictionary already exists")
