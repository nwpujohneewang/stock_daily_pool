package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"stock/internal/cache"
	"stock/internal/config"
	"stock/internal/external/llm"
	"stock/internal/external/tushare"
	"stock/internal/repo"
	"stock/internal/ws"
)

type Handlers struct {
	DB         *gorm.DB
	Redis      *redis.Client
	Tushare    *tushare.Client
	LLM        *llm.Client
	Hub        *ws.Hub
	QuoteCache *cache.QuoteCache
	PoolCache  *cache.PoolCache
	Config     *config.AppConfig

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
	db *gorm.DB,
	redisClient *redis.Client,
	tushareClient *tushare.Client,
	llmClient *llm.Client,
	hub *ws.Hub,
	quoteCache *cache.QuoteCache,
	poolCache *cache.PoolCache,
	cfg *config.AppConfig,
) *Handlers {
	topicRepo := repo.NewTopicRepo(db)
	mappingRepo := repo.NewMappingRepo(db)
	alertRepo := repo.NewAlertRepo(db)
	poolRepo := repo.NewPoolRepo(db)
	boardRepo := repo.NewBoardRepo(db)
	synonymRepo := repo.NewSynonymRepo(db)
	evidenceRepo := repo.NewEvidenceRepo(db)
	conceptRepo := repo.NewConceptRepo(db)
	conceptDetailRepo := repo.NewConceptDetailRepo(db)

	focusCache := cache.NewFocusCache(redisClient)

	h := &Handlers{
		DB:         db,
		Redis:      redisClient,
		Tushare:    tushareClient,
		LLM:        llmClient,
		Hub:        hub,
		QuoteCache: quoteCache,
		PoolCache:  poolCache,
		Config:     cfg,
	}

	h.PoolHandler = *NewPoolHandler(poolRepo, alertRepo)
	h.TopicHandler = *NewTopicHandler(topicRepo, mappingRepo, boardRepo)
	h.AlertHandler = *NewAlertHandler(alertRepo)
	h.SynonymHandler = *NewSynonymHandler(synonymRepo, topicRepo)
	h.FocusHandler = *NewFocusHandler(focusCache, topicRepo)
	h.EvidenceHandler = *NewEvidenceHandler(evidenceRepo)
	h.ConceptHandler = *NewConceptHandler(conceptRepo, conceptDetailRepo, mappingRepo, topicRepo)
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

type PoolHandler struct {
	poolRepo  *repo.PoolRepo
	alertRepo *repo.AlertRepo
}

func NewPoolHandler(poolRepo *repo.PoolRepo, alertRepo *repo.AlertRepo) *PoolHandler {
	return &PoolHandler{poolRepo: poolRepo, alertRepo: alertRepo}
}

func (h *PoolHandler) GetLimitUp(c *gin.Context) {
	if h.poolRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
	ctx := c.Request.Context()
	date := c.Query("date")
	if date == "" {
		date = "2026-03-18"
	}

	pools, err := h.poolRepo.GetByDate(ctx, date, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(pools))
}

func (h *PoolHandler) GetAbove5(c *gin.Context) {
	if h.poolRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
	ctx := c.Request.Context()
	date := c.Query("date")
	if date == "" {
		date = "2026-03-18"
	}

	pools, err := h.poolRepo.GetByDate(ctx, date, 2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(pools))
}

type TopicHandler struct {
	topicRepo   *repo.TopicRepo
	mappingRepo *repo.MappingRepo
	boardRepo   *repo.BoardRepo
}

func NewTopicHandler(topicRepo *repo.TopicRepo, mappingRepo *repo.MappingRepo, boardRepo *repo.BoardRepo) *TopicHandler {
	return &TopicHandler{topicRepo: topicRepo, mappingRepo: mappingRepo, boardRepo: boardRepo}
}

func (h *TopicHandler) List(c *gin.Context) {
	if h.topicRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
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

	topics, total, err := h.topicRepo.List(ctx, keyword, page, pageSize)
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

type AlertHandler struct {
	alertRepo *repo.AlertRepo
}

func NewAlertHandler(alertRepo *repo.AlertRepo) *AlertHandler {
	return &AlertHandler{alertRepo: alertRepo}
}

func (h *AlertHandler) GetTodayAlerts(c *gin.Context) {
	if h.alertRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
	ctx := c.Request.Context()
	date := c.Query("date")
	if date == "" {
		date = "2026-03-18"
	}

	alerts, err := h.alertRepo.GetTodayAlerts(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(alerts))
}
