package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStockRepo(t *testing.T) {
	repo := NewStockRepository()
	assert.NotNil(t, repo)
}

func TestNewTopicRepo(t *testing.T) {
	repo := NewTopicRepository()
	assert.NotNil(t, repo)
}

func TestNewAlertRepo(t *testing.T) {
	repo := NewAlertRepository()
	assert.NotNil(t, repo)
}

func TestNewPoolRepo(t *testing.T) {
	repo := NewPoolRepository()
	assert.NotNil(t, repo)
}

func TestNewMappingRepo(t *testing.T) {
	repo := NewMappingRepository()
	assert.NotNil(t, repo)
}

func TestNewBoardRepo(t *testing.T) {
	repo := NewBoardRepository()
	assert.NotNil(t, repo)
}

func TestNewSynonymRepo(t *testing.T) {
	repo := NewSynonymRepository()
	assert.NotNil(t, repo)
}

func TestNewEvidenceRepo(t *testing.T) {
	repo := NewEvidenceRepository()
	assert.NotNil(t, repo)
}

func TestNewConceptRepo(t *testing.T) {
	repo := NewConceptRepository()
	assert.NotNil(t, repo)
}

func TestNewConceptDetailRepo(t *testing.T) {
	repo := NewConceptDetailRepository()
	assert.NotNil(t, repo)
}

func TestNewMarketSnapshotRepo(t *testing.T) {
	repo := NewMarketSnapshotRepository()
	assert.NotNil(t, repo)
}
