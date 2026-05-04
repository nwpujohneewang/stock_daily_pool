package response

// NewsItem 单条新闻
type NewsItem struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Source      string `json:"source"`
	Importance  string `json:"importance"`
	PublishedAt string `json:"published_at"`
}

// FetchNewsResp GET /hot-spot/fetch-news 响应
type FetchNewsResp struct {
	Total int        `json:"total"`
	News  []NewsItem `json:"news"`
}
