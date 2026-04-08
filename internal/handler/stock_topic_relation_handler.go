package handler

import (
	"errors"
	"net/http"
	"strconv"

	"stock/internal/service/topic"
	"stock/model/api"
	"stock/model/api/response"
	"stock/model/dal_model"

	"github.com/gin-gonic/gin"
)

type StockTopicRelationHandler struct{}

func NewStockTopicRelationHandler() *StockTopicRelationHandler {
	return &StockTopicRelationHandler{}
}

func stockTopicRelationItems(relations []dal_model.StockTopicRelation) []response.TopicRelationItem {
	items := make([]response.TopicRelationItem, 0, len(relations))
	for _, relation := range relations {
		items = append(items, response.TopicRelationItem{
			TopicID:       relation.TopicID,
			TopicName:     relation.TopicName,
			Category:      relation.Category,
			Source:        relation.Source,
			HitCount:      relation.HitCount,
			FirstSeenDate: relation.FirstSeenDate,
			LastSeenDate:  relation.LastSeenDate,
		})
	}
	return items
}

func (h *StockTopicRelationHandler) AddManual(c *gin.Context) {
	var req struct {
		TopicID int64 `json:"topic_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TopicID <= 0 {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid topic id"))
		return
	}

	relations, err := topic.NewStockTopicRelationService().AddManualRelation(c.Request.Context(), c.Param("ts_code"), req.TopicID)
	if err != nil {
		if errors.Is(err, topic.ErrStockTopicRelationTopicNotFound) {
			c.JSON(http.StatusNotFound, api.Fail(404, "topic not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, api.Fail(500, "add manual topic relation failed"))
		return
	}
	c.JSON(http.StatusOK, api.OK(stockTopicRelationItems(relations)))
}

func (h *StockTopicRelationHandler) Delete(c *gin.Context) {
	topicID, err := strconv.ParseInt(c.Param("topic_id"), 10, 64)
	if err != nil || topicID <= 0 {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid topic id"))
		return
	}

	relations, err := topic.NewStockTopicRelationService().DeleteRelation(c.Request.Context(), c.Param("ts_code"), topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "delete stock topic relation failed"))
		return
	}
	c.JSON(http.StatusOK, api.OK(stockTopicRelationItems(relations)))
}
