package handler

import (
	"net/http"
	"stock/dal/db"
	"stock/internal/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SynonymHandler struct{}

func NewSynonymHandler() *SynonymHandler {
	return &SynonymHandler{}
}

func (h *SynonymHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	topicID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid topic id"))
		return
	}

	synonyms, err := db.NewSynonymRepository().GetByTopicID(ctx, topicID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(synonyms))
}

func (h *SynonymHandler) Create(c *gin.Context) {
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

	synonym := model.TopicSynonym{
		TopicID: topicID,
		Synonym: req.Synonym,
		Source:  "manual",
	}

	created, err := db.NewSynonymRepository().Create(ctx, synonym)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, OK(created))
}

func (h *SynonymHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	synonymID, err := strconv.ParseInt(c.Param("synonym_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid synonym id"))
		return
	}

	if err := db.NewSynonymRepository().Delete(ctx, synonymID); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(nil))
}
