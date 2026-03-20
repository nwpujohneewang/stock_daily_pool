package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQuoteCache(t *testing.T) {
	cache := &QuoteCache{}
	assert.NotNil(t, cache)
}

func TestNewPoolCache(t *testing.T) {
	cache := &PoolCache{}
	assert.NotNil(t, cache)
}

func TestNewFocusCache(t *testing.T) {
	cache := &FocusCache{}
	assert.NotNil(t, cache)
}

func TestNewMappingCache(t *testing.T) {
	cache := &MappingCache{}
	assert.NotNil(t, cache)
}

func TestNewConceptCache(t *testing.T) {
	cache := &ConceptCache{}
	assert.NotNil(t, cache)
}

func TestNewActivityCache(t *testing.T) {
	cache := &ActivityCache{}
	assert.NotNil(t, cache)
}

func TestNewAlertDedupCache(t *testing.T) {
	cache := &AlertDedupCache{}
	assert.NotNil(t, cache)
}

func TestNewSnapshotCache(t *testing.T) {
	cache := &SnapshotCache{}
	assert.NotNil(t, cache)
}
