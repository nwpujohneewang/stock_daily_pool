package handler

import (
	"net/http"
	"stock/internal/service/topic"
	"stock/model/api"
	"stock/model/api/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TopicHandler struct{}

func NewTopicHandler() *TopicHandler {
	return &TopicHandler{}
}

func (h *TopicHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	svc := topic.NewTopicQueryService()
	result, err := svc.List(ctx, topic.TopicQueryParams{
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "failed to list topics"))
		return
	}

	items := make([]response.TopicItem, 0, len(result.Items))
	for _, t := range result.Items {
		items = append(items, response.TopicItem{
			ID:              t.ID,
			Name:            t.Name,
			Category:        t.Category,
			Source:          t.Source,
			OccurrenceCount: t.OccurrenceCount,
			FirstSeenDate:   t.FirstSeenDate,
			LastSeenDate:    t.LastSeenDate,
		})
	}

	pages := (result.Total + int64(pageSize) - 1) / int64(pageSize)
	c.JSON(http.StatusOK, api.OK(response.TopicListResp{
		Items: items,
		Total: result.Total,
		Page:  page,
		Pages: pages,
	}))
}
