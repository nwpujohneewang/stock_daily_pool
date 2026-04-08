// internal/service/stock_query_service.go
package stock

import (
	"context"
	"fmt"

	"stock/dal/repo"
)

type StockQueryServiceImpl struct{}

func NewStockQueryService() *StockQueryServiceImpl {
	return &StockQueryServiceImpl{}
}

func (s *StockQueryServiceImpl) Search(ctx context.Context, params StockSearchParams) (*StockSearchResult, error) {
	stockRepo := repo.NewStockRepository()

	stocks, total, err := stockRepo.GetPaginatedStocks(ctx, params.Query, params.Topic, params.Category, params.Page, params.PageSize)
	if err != nil {
		return nil, fmt.Errorf("search stocks: %w", err)
	}

	items := make([]StockResult, 0, len(stocks))
	for _, stock := range stocks {
		items = append(items, StockResult{
			TsCode:    stock.TsCode,
			Symbol:    stock.Symbol,
			Name:      stock.Name,
			Exchange:  stock.Exchange,
			BoardCode: stock.BoardCode,
			Industry:  stock.Industry,
			IsST:      stock.IsST,
			ListDate:  stock.ListDate,
		})
	}

	return &StockSearchResult{
		Items: items,
		Total: total,
	}, nil
}

func (s *StockQueryServiceImpl) GetDetail(ctx context.Context, tsCode string) (*StockDetailResult, error) {
	stockRepo := repo.NewStockRepository()
	relationRepo := repo.NewStockTopicRelationRepository()

	stock, err := stockRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, fmt.Errorf("get stock: %w", err)
	}
	if stock == nil {
		return nil, ErrStockNotFound
	}

	relations, err := relationRepo.GetByTsCode(ctx, tsCode)
	if err != nil {
		return nil, fmt.Errorf("get stock topics: %w", err)
	}

	topics := make([]TopicRelationResult, 0, len(relations))
	for _, r := range relations {
		topics = append(topics, TopicRelationResult{
			TopicID:       r.TopicID,
			TopicName:     r.TopicName,
			Category:      r.Category,
			Source:        r.Source,
			HitCount:      r.HitCount,
			FirstSeenDate: r.FirstSeenDate,
			LastSeenDate:  r.LastSeenDate,
		})
	}

	return &StockDetailResult{
		Info: StockResult{
			TsCode:    stock.TsCode,
			Symbol:    stock.Symbol,
			Name:      stock.Name,
			Exchange:  stock.Exchange,
			BoardCode: stock.BoardCode,
			Industry:  stock.Industry,
			IsST:      stock.IsST,
			ListDate:  stock.ListDate,
		},
		Topics: topics,
	}, nil
}
