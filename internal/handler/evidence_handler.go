package handler

import (
	"net/http"
	"stock/dal/db"
	"strconv"

	"github.com/gin-gonic/gin"
)

type EvidenceHandler struct{}

func NewEvidenceHandler() *EvidenceHandler {
	return &EvidenceHandler{}
}

func (h *EvidenceHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	tsCode := c.Param("ts_code")
	date := c.Query("date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	repo := db.NewEvidenceRepository()

	if tsCode != "" {
		evidences, err := repo.GetByStock(ctx, tsCode, date)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, OK(evidences))
		return
	}

	result, err := repo.List(ctx, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(result))
}

func (h *EvidenceHandler) Correct(c *gin.Context) {
	ctx := c.Request.Context()
	c.Param("ts_code")
	evidenceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid evidence id"))
		return
	}

	var req struct {
		CorrectedTopicID int64 `json:"corrected_topic_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Fail(400, "invalid request body"))
		return
	}

	if req.CorrectedTopicID == 0 {
		c.JSON(http.StatusBadRequest, Fail(400, "corrected_topic_id is required"))
		return
	}

	repo := db.NewEvidenceRepository()

	_, err = repo.GetByID(ctx, evidenceID)
	if err != nil {
		c.JSON(http.StatusNotFound, Fail(404, "evidence not found"))
		return
	}

	if err := repo.UpdateCorrectedTopicID(ctx, evidenceID, req.CorrectedTopicID); err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(nil))
}
