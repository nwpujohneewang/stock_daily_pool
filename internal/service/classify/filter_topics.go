package classify

var FilteredTopicNames = map[string]bool{
	"公告":    true,
	"其他":    true,
	"新股":    true,
	"ST板块":  true,
	"次新":    true,
	"退市整理":  true,
	"龙字辈":   true,
	"谐音":    true,
	"马字辈":   true,
	"凤字辈":   true,
	"华字辈":   true,
	"好名字":   true,
	"周杰伦":   true,
	"深圳国资":  true,
	"央企国资":  true,
	"湖北国资":  true,
	"上海国资":  true,
	"中字头":   true,
	"国改/重组": true,
	"涨价收益":  true,
	"涨价":    true,
	"东盟":    true,
	"新质生产力": true,
}

func IsFilteredTopic(topicName string) bool {
	return FilteredTopicNames[topicName]
}
