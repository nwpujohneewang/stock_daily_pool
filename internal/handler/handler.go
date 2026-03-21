package handler

import (
	"net/http"
	"stock/dal/db"
	"strconv"

	"github.com/gin-gonic/gin"
	"stock/config"
	"stock/internal/ws"
)

type Handlers struct {
	Hub    *ws.Hub
	Config *config.AppConfig

	PoolHandler
	TopicHandler
	AlertHandler
	SynonymHandler
	FocusHandler
	EvidenceHandler
	ConceptHandler
	WSHandler
}

func NewHandlers(
	hub *ws.Hub,
	cfg *config.AppConfig,
) *Handlers {
	h := &Handlers{
		Hub:    hub,
		Config: cfg,
	}

	h.PoolHandler = *NewPoolHandler()
	h.TopicHandler = *NewTopicHandler()
	h.AlertHandler = *NewAlertHandler()
	h.SynonymHandler = *NewSynonymHandler()
	h.FocusHandler = *NewFocusHandler()
	h.EvidenceHandler = *NewEvidenceHandler()
	h.ConceptHandler = *NewConceptHandler()
	h.WSHandler = *NewWSHandler(hub)

	return h
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(data interface{}) Response {
	return Response{Code: 0, Message: "ok", Data: data}
}

func Fail(code int, message string) Response {
	return Response{Code: code, Message: message}
}

type PoolHandler struct{}

func NewPoolHandler() *PoolHandler {
	return &PoolHandler{}
}

func (h *PoolHandler) GetLimitUp(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", "2026-03-18")

	pools, err := db.NewPoolRepository().GetByDate(ctx, date, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(pools))
}

func (h *PoolHandler) GetAbove5(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", "2026-03-18")

	pools, err := db.NewPoolRepository().GetByDate(ctx, date, 2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(pools))
}

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

	topics, total, err := db.NewTopicRepository().List(ctx, keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(map[string]interface{}{
		"items": topics,
		"total": total,
		"page":  page,
		"pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}))
}

type AlertHandler struct{}

func NewAlertHandler() *AlertHandler {
	return &AlertHandler{}
}

func (h *AlertHandler) GetTodayAlerts(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", "2026-03-18")

	alerts, err := db.NewAlertRepository().GetTodayAlerts(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(alerts))
}
