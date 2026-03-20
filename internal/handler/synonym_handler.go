package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"stock/internal/repo"
)

type SynonymHandler struct {
	synonymRepo *repo.SynonymRepo
	topicRepo   *repo.TopicRepo
}

func NewSynonymHandler(synonymRepo *repo.SynonymRepo, topicRepo *repo.TopicRepo) *SynonymHandler {
	return &SynonymHandler{synonymRepo: synonymRepo, topicRepo: topicRepo}
}

func (h *SynonymHandler) List(c *gin.Context) {
	if h.synonymRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
	ctx := c.Request.Context()
	topicID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid topic id"))
		return
	}

	synonyms, err := h.synonymRepo.GetByTopicID(ctx, topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(synonyms))
}

func (h *SynonymHandler) Create(c *gin.Context) {
	if h.synonymRepo == nil || h.topicRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
	ctx := c.Request.Context()
	topicID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid topic id"))
		return
	}

	var req struct {
		Synonym string `json:"synonym"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid request body"))
		return
	}

	if req.Synonym == "" {
		c.JSON(http.StatusBadRequest, Fail(400, "synonym is required"))
		return
	}

	synonym := repo.TopicSynonym{
		TopicID: topicID,
		Synonym: req.Synonym,
		Source:  "manual",
	}

	created, err := h.synonymRepo.Create(ctx, synonym)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, OK(created))
}

func (h *SynonymHandler) Delete(c *gin.Context) {
	if h.synonymRepo == nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "repo not initialized"))
		return
	}
	ctx := c.Request.Context()
	synonymID, err := strconv.ParseInt(c.Param("synonym_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid synonym id"))
		return
	}

	if err := h.synonymRepo.Delete(ctx, synonymID); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(nil))
}
