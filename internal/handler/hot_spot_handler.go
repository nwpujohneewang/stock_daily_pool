package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"stock/external/news"
	"stock/model/api"
	"stock/model/api/response"
)

type HotSpotHandler struct {
	newsAgg *news.Aggregator
}

func NewHotSpotHandler(agg *news.Aggregator) *HotSpotHandler {
	return &HotSpotHandler{newsAgg: agg}
}

// FetchNews GET /api/v1/hot-spot/fetch-news?source=all|cls|eastmoney
func (h *HotSpotHandler) FetchNews(c *gin.Context) {
	if h.newsAgg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "news aggregator not configured"})
		return
	}

	source := c.DefaultQuery("source", "all")
	minImportance := c.DefaultQuery("importance", "medium")

	importanceToCLSLevel := map[string]string{"high": "A", "medium": "B", "low": ""}
	clsMinLevel := importanceToCLSLevel[minImportance]
	if clsMinLevel == "" && minImportance != "low" {
		clsMinLevel = "B"
	}

	newsList, err := h.newsAgg.FetchAll(c.Request.Context(), clsMinLevel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]response.NewsItem, 0, len(newsList))
	for _, n := range newsList {
		if source != "all" && n.Source != source {
			continue
		}
		items = append(items, response.NewsItem{
			Title:       n.Title,
			Content:     n.Content,
			Source:      n.Source,
			Importance:  n.Importance,
			PublishedAt: n.PublishTime.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, api.OK(response.FetchNewsResp{Total: len(items), News: items}))
}
