// internal/handler/topic_dict_handler.go
package handler

import (
	"net/http"
	"stock/internal/service/topic"
	"strconv"

	"stock/model/api"
	"stock/model/api/request"
	"stock/model/api/response"

	"github.com/gin-gonic/gin"
)

type TopicDictionaryHandler struct{}

func NewTopicDictionaryHandler() *TopicDictionaryHandler {
	return &TopicDictionaryHandler{}
}

func (h *TopicDictionaryHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	svc := topic.NewTopicDictService()

	results, err := svc.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "failed to list topic dictionaries"))
		return
	}

	items := make([]response.TopicDictItem, 0, len(results))
	for _, r := range results {
		items = append(items, response.TopicDictItem{
			ID:             r.ID,
			RawTopicName:   r.RawTopicName,
			NormalizedName: r.NormalizedName,
			Category:       r.Category,
		})
	}

	c.JSON(http.StatusOK, api.OK(items))
}

func (h *TopicDictionaryHandler) Create(c *gin.Context) {
	var req request.CreateTopicDictReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid request: "+err.Error()))
		return
	}

	ctx := c.Request.Context()
	svc := topic.NewTopicDictService()

	result, err := svc.Create(ctx, topic.TopicDictParams{
		RawTopicName:   req.RawTopicName,
		NormalizedName: req.NormalizedName,
		Category:       req.Category,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "create topic dictionary failed"))
		return
	}

	c.JSON(http.StatusCreated, api.Created(response.TopicDictItem{
		ID:             result.ID,
		RawTopicName:   result.RawTopicName,
		NormalizedName: result.NormalizedName,
		Category:       result.Category,
	}))
}

func (h *TopicDictionaryHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid id"))
		return
	}

	var req request.UpdateTopicDictReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid request: "+err.Error()))
		return
	}

	ctx := c.Request.Context()
	svc := topic.NewTopicDictService()

	if err := svc.Update(ctx, id, topic.TopicDictParams{
		RawTopicName:   req.RawTopicName,
		NormalizedName: req.NormalizedName,
		Category:       req.Category,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "update topic dictionary failed"))
		return
	}

	c.JSON(http.StatusOK, api.OK(nil))
}

func (h *TopicDictionaryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid id"))
		return
	}

	ctx := c.Request.Context()
	svc := topic.NewTopicDictService()

	if err := svc.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "delete topic dictionary failed"))
		return
	}

	c.JSON(http.StatusOK, api.OK(nil))
}
