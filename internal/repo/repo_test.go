package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStockRepo(t *testing.T) {
	repo := &StockRepo{}
	assert.NotNil(t, repo)
}

func TestNewTopicRepo(t *testing.T) {
	repo := &TopicRepo{}
	assert.NotNil(t, repo)
}

func TestNewAlertRepo(t *testing.T) {
	repo := &AlertRepo{}
	assert.NotNil(t, repo)
}

func TestNewPoolRepo(t *testing.T) {
	repo := &PoolRepo{}
	assert.NotNil(t, repo)
}

func TestNewMappingRepo(t *testing.T) {
	repo := &MappingRepo{}
	assert.NotNil(t, repo)
}

func TestNewBoardRepo(t *testing.T) {
	repo := &BoardRepo{}
	assert.NotNil(t, repo)
}

func TestNewSynonymRepo(t *testing.T) {
	repo := &SynonymRepo{}
	assert.NotNil(t, repo)
}

func TestNewEvidenceRepo(t *testing.T) {
	repo := &EvidenceRepo{}
	assert.NotNil(t, repo)
}

func TestNewConceptRepo(t *testing.T) {
	repo := &ConceptRepo{}
	assert.NotNil(t, repo)
}

func TestNewConceptDetailRepo(t *testing.T) {
	repo := &ConceptDetailRepo{}
	assert.NotNil(t, repo)
}

func TestNewMarketSnapshotRepo(t *testing.T) {
	repo := &MarketSnapshotRepo{}
	assert.NotNil(t, repo)
}
