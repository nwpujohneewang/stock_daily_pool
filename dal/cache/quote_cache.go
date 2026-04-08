package cache

import (
	"context"
	"stock/model/dal_model"
)

type QuoteCacheInterface interface {
	Get(ctx context.Context, tsCode string) (*dal_model.StockQuote, error)
	GetBatch(ctx context.Context, tsCodes []string) (map[string]*dal_model.StockQuote, error)
	Set(ctx context.Context, quote *dal_model.StockQuote) error
	SetBatch(ctx context.Context, quotes []*dal_model.StockQuote) error
	Delete(ctx context.Context, tsCode string) error
	GetAll(ctx context.Context) ([]*dal_model.StockQuote, error)
}

var _ QuoteCacheInterface = (*QuoteCacheImpl)(nil)

type QuoteCacheImpl struct{}

func NewQuoteCache() *QuoteCacheImpl {
	return &QuoteCacheImpl{}
}

const quoteCacheKey = "rt:quotes"

func (c QuoteCacheImpl) Get(ctx context.Context, tsCode string) (*dal_model.StockQuote, error) {
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(quoteCacheKey); found {
		quotes := v.(map[string]*dal_model.StockQuote)
		if quote, exists := quotes[tsCode]; exists {
			// Return a copy
			copied := *quote
			return &copied, nil
		}
	}
	return nil, nil
}

func (c QuoteCacheImpl) GetBatch(ctx context.Context, tsCodes []string) (map[string]*dal_model.StockQuote, error) {
	if len(tsCodes) == 0 {
		return nil, nil
	}
	RLock()
	defer RUnlock()

	result := make(map[string]*dal_model.StockQuote, len(tsCodes))
	if v, found := Cache.Get(quoteCacheKey); found {
		quotes := v.(map[string]*dal_model.StockQuote)
		for _, tc := range tsCodes {
			if quote, exists := quotes[tc]; exists {
				copied := *quote
				result[tc] = &copied
			}
		}
	}
	return result, nil
}

func (c QuoteCacheImpl) Set(ctx context.Context, quote *dal_model.StockQuote) error {
	Lock()
	defer Unlock()

	var quotes map[string]*dal_model.StockQuote
	if v, found := Cache.Get(quoteCacheKey); found {
		quotes = v.(map[string]*dal_model.StockQuote)
	} else {
		quotes = make(map[string]*dal_model.StockQuote)
	}
	// Store a copy
	copied := *quote
	quotes[quote.TsCode] = &copied
	Cache.Set(quoteCacheKey, quotes, TTLUntilEndOfDay())
	return nil
}

func (c QuoteCacheImpl) SetBatch(ctx context.Context, quotes []*dal_model.StockQuote) error {
	if len(quotes) == 0 {
		return nil
	}
	Lock()
	defer Unlock()

	var existing map[string]*dal_model.StockQuote
	if v, found := Cache.Get(quoteCacheKey); found {
		existing = v.(map[string]*dal_model.StockQuote)
	} else {
		existing = make(map[string]*dal_model.StockQuote)
	}
	for _, quote := range quotes {
		if quote == nil {
			continue
		}
		copied := *quote
		existing[quote.TsCode] = &copied
	}
	Cache.Set(quoteCacheKey, existing, TTLUntilEndOfDay())
	return nil
}

func (c QuoteCacheImpl) Delete(ctx context.Context, tsCode string) error {
	Lock()
	defer Unlock()

	if v, found := Cache.Get(quoteCacheKey); found {
		quotes := v.(map[string]*dal_model.StockQuote)
		delete(quotes, tsCode)
		Cache.Set(quoteCacheKey, quotes, TTLUntilEndOfDay())
	}
	return nil
}

func (c QuoteCacheImpl) GetAll(ctx context.Context) ([]*dal_model.StockQuote, error) {
	RLock()
	defer RUnlock()

	if v, found := Cache.Get(quoteCacheKey); found {
		quotes := v.(map[string]*dal_model.StockQuote)
		result := make([]*dal_model.StockQuote, 0, len(quotes))
		for _, quote := range quotes {
			// Return copies
			copied := *quote
			result = append(result, &copied)
		}
		return result, nil
	}
	return []*dal_model.StockQuote{}, nil
}
