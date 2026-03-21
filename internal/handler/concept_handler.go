package handler

import (
	"net/http"
	"stock/dal/db"
	"stock/model/dal_model"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ConceptHandler struct{}

func NewConceptHandler() *ConceptHandler {
	return &ConceptHandler{}
}

func (h *ConceptHandler) ListByTopic(c *gin.Context) {
	ctx := c.Request.Context()
	topicID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid topic id"))
		return
	}

	mappings, err := db.NewMappingRepository().GetConceptMappingsByTopic(ctx, topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(mappings))
}

func (h *ConceptHandler) ListMappings(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	conceptRepo := db.NewConceptRepository()
	mappingRepo := db.NewMappingRepository()
	topicRepo := db.NewTopicRepository()

	concepts, total, err := conceptRepo.List(ctx, keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	type ConceptWithMapping struct {
		dal_model.TushareConcept
		IsMapped  bool   `json:"is_mapped"`
		TopicID   *int64 `json:"topic_id,omitempty"`
		TopicName string `json:"topic_name,omitempty"`
	}

	var results []ConceptWithMapping
	for _, concept := range concepts {
		mapping, err := mappingRepo.GetConceptMapping(ctx, concept.ConceptName)
		cm := ConceptWithMapping{TushareConcept: concept}
		if err == nil && mapping != nil {
			cm.IsMapped = true
			cm.TopicID = &mapping.TopicID
			topic, err := topicRepo.GetByID(ctx, mapping.TopicID)
			if err == nil && topic != nil {
				cm.TopicName = topic.Name
			}
		}
		results = append(results, cm)
	}

	c.JSON(http.StatusOK, OK(map[string]interface{}{
		"items": results,
		"total": total,
		"page":  page,
	}))
}

func (h *ConceptHandler) CreateMapping(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		ConceptName string `json:"concept_name"`
		ConceptCode string `json:"concept_code"`
		TopicID     int64  `json:"topic_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid request body"))
		return
	}

	if err := db.NewMappingRepository().CreateConceptMapping(ctx, req.ConceptName, req.ConceptCode, req.TopicID, "manual"); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, OK(nil))
}
