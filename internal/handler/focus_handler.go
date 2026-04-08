// internal/handler/focus_handler.go
package handler

import (
	"net/http"
	"stock/internal/service/focus"
	"strconv"
	"time"

	"stock/dal/dao"
	"stock/model/api"
	"stock/model/api/request"
	"stock/model/api/response"

	"github.com/gin-gonic/gin"
)

type FocusHandler struct{}

func NewFocusHandler() *FocusHandler {
	return &FocusHandler{}
}

func (h *FocusHandler) Get(c *gin.Context) {
	date := c.DefaultQuery("date", dao.Now().Format("2006-01-02"))

	ctx := c.Request.Context()
	svc := focus.NewFocusService()

	results, err := svc.Get(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, err.Error()))
		return
	}

	items := make([]response.FocusTopicItem, 0, len(results))
	for _, r := range results {
		items = append(items, response.FocusTopicItem{
			ID:              r.ID,
			Name:            r.Name,
			Category:        r.Category,
			Source:          r.Source,
			OccurrenceCount: r.OccurrenceCount,
			FirstSeenDate:   r.FirstSeenDate,
			LastSeenDate:    r.LastSeenDate,
		})
	}

	c.JSON(http.StatusOK, api.OK(items))
}

func (h *FocusHandler) Set(c *gin.Context) {
	var req request.SetFocusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid request: "+err.Error()))
		return
	}

	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}

	ctx := c.Request.Context()
	svc := focus.NewFocusService()

	if err := svc.Set(ctx, req.Date, req.TopicIDs); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.OK(nil))
}

func (h *FocusHandler) Delete(c *gin.Context) {
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	topicID, err := strconv.ParseInt(c.Param("topic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid topic id"))
		return
	}

	ctx := c.Request.Context()
	svc := focus.NewFocusService()

	if err := svc.Delete(ctx, date, topicID); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.OK(nil))
}
