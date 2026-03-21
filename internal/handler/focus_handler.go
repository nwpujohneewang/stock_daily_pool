package handler

import (
	"net/http"
	"stock/dal/db"
	"stock/dal/redis"
	"stock/model/dal_model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type FocusHandler struct{}

func NewFocusHandler() *FocusHandler {
	return &FocusHandler{}
}

func (h *FocusHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	focusCache := redis.NewFocusCache()
	topicIDs, err := focusCache.GetFocusTopics(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	var topics []dal_model.Topic
	for _, id := range topicIDs {
		topic, err := db.NewTopicRepository().GetByID(ctx, id)
		if err != nil {
			continue
		}
		topics = append(topics, *topic)
	}

	c.JSON(http.StatusOK, OK(topics))
}

func (h *FocusHandler) Set(c *gin.Context) {
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

	focusCache := redis.NewFocusCache()
	if err := focusCache.SetFocusTopics(ctx, req.Date, req.TopicIDs); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(nil))
}

func (h *FocusHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	topicIDInt, err := strconv.ParseInt(c.Param("topic_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid topic id"))
		return
	}

	focusCache := redis.NewFocusCache()
	if err := focusCache.RemoveFocusTopic(ctx, date, topicIDInt); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(nil))
}
