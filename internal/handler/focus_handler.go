package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"stock/internal/cache"
	"stock/internal/model"
	"stock/internal/repo"
)

type FocusHandler struct {
	focusCache *cache.FocusCache
	topicRepo  *repo.TopicRepo
}

func NewFocusHandler(focusCache *cache.FocusCache, topicRepo *repo.TopicRepo) *FocusHandler {
	return &FocusHandler{focusCache: focusCache, topicRepo: topicRepo}
}

func (h *FocusHandler) Get(c *gin.Context) {
	if h.focusCache == nil || h.topicRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
	ctx := c.Request.Context()
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	topicIDs, err := h.focusCache.GetFocusTopics(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	var topics []model.Topic
	for _, id := range topicIDs {
		topic, err := h.topicRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		topics = append(topics, *topic)
	}

	c.JSON(http.StatusOK, OK(topics))
}

func (h *FocusHandler) Set(c *gin.Context) {
	if h.focusCache == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "cache not initialized"))
		return
	}
	ctx := c.Request.Context()
	var req struct {
		Date     string  `json:"date"`
		TopicIDs []int64 `json:"topic_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid request body"))
		return
	}

	if req.Date == "" {
		req.Date = time.Now().Format("2006-01-02")
	}

	if err := h.focusCache.SetFocusTopics(ctx, req.Date, req.TopicIDs); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(nil))
}

func (h *FocusHandler) Delete(c *gin.Context) {
	if h.focusCache == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "cache not initialized"))
		return
	}
	ctx := c.Request.Context()
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	topicIDInt, err := strconv.ParseInt(c.Param("topic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid topic id"))
		return
	}

	if err := h.focusCache.RemoveFocusTopic(ctx, date, topicIDInt); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(nil))
}
