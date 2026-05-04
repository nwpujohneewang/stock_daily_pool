package handler

import (
	"net/http"
	"stock/config"
	"stock/external/tushare"
	"stock/internal/service/stock"
	"stock/model/api"
	"stock/model/api/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type StockHandler struct{}

func NewStockHandler() *StockHandler {
	return &StockHandler{}
}

func (h *StockHandler) Search(c *gin.Context) {
	ctx := c.Request.Context()
	q := c.Query("q")
	topicName := c.Query("topic")
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	svc := stock.NewStockQueryService()
	result, err := svc.Search(ctx, stock.StockSearchParams{
		Query:    q,
		Topic:    topicName,
		Category: category,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "search stocks failed: "+err.Error()))
		return
	}

	items := make([]response.StockItem, 0, len(result.Items))
	for _, s := range result.Items {
		items = append(items, response.StockItem{
			TsCode:    s.TsCode,
			Symbol:    s.Symbol,
			Name:      s.Name,
			Exchange:  s.Exchange,
			BoardCode: s.BoardCode,
			Industry:  s.Industry,
			IsST:      s.IsST,
			ListDate:  s.ListDate,
		})
	}

	c.JSON(http.StatusOK, api.OK(response.StockSearchResp{
		Items:    items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	}))
}

func (h *StockHandler) GetDetail(c *gin.Context) {
	ctx := c.Request.Context()
	tsCode := c.Param("ts_code")

	svc := stock.NewStockQueryService()
	result, err := svc.GetDetail(ctx, tsCode)
	if err != nil {
		if err == stock.ErrStockNotFound {
			c.JSON(http.StatusNotFound, api.Fail(404, "stock not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, api.Fail(500, "get stock detail failed: "+err.Error()))
		return
	}

	topics := make([]response.TopicRelationItem, 0, len(result.Topics))
	for _, t := range result.Topics {
		topics = append(topics, response.TopicRelationItem{
			TopicID:       t.TopicID,
			TopicName:     t.TopicName,
			Category:      t.Category,
			Source:        t.Source,
			HitCount:      t.HitCount,
			FirstSeenDate: t.FirstSeenDate,
			LastSeenDate:  t.LastSeenDate,
		})
	}

	c.JSON(http.StatusOK, api.OK(response.StockDetail{
		Info: response.StockItem{
			TsCode:    result.Info.TsCode,
			Symbol:    result.Info.Symbol,
			Name:      result.Info.Name,
			Exchange:  result.Info.Exchange,
			BoardCode: result.Info.BoardCode,
			Industry:  result.Info.Industry,
			IsST:      result.Info.IsST,
			ListDate:  result.Info.ListDate,
		},
		Topics: topics,
	}))
}

type SyncBasicRequest struct {
	Date string `json:"date"`
}

func (h *StockHandler) SyncBasic(c *gin.Context) {
	var req SyncBasicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		req.Date = time.Now().In(loc).Format("2006-01-02")
	}
	if req.Date == "" {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		req.Date = time.Now().In(loc).Format("2006-01-02")
	}

	cfg := config.GlobalConfig
	tushareClient := tushare.NewClient(&cfg.Tushare, cfg.Retry)
	svc := stock.NewStockService(tushareClient)

	if err := svc.SyncStockBasic(c.Request.Context(), req.Date); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "sync stock basic failed: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.OK(gin.H{
		"message": "sync completed",
		"date":    req.Date,
	}))
}
