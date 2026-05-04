package news

import (
	"context"
	"stock/model/core_model"

	"go.uber.org/zap"
)

// Aggregator combines multiple news source clients into a single interface.
type Aggregator struct {
	cls       *CLSClient
	eastMoney *EastMoneyClient
	logger    *zap.Logger
}

func NewAggregator(cls *CLSClient, eastMoney *EastMoneyClient, logger *zap.Logger) *Aggregator {
	return &Aggregator{cls: cls, eastMoney: eastMoney, logger: logger}
}

// FetchAll fetches latest news from all sources.
func (a *Aggregator) FetchAll(ctx context.Context, minLevel string) ([]core_model.RawNews, error) {
	var all []core_model.RawNews
	if a.cls != nil {
		items, err := a.cls.FetchLatest(ctx, 50, minLevel)
		if err != nil {
			a.logger.Warn("cls fetch failed", zap.Error(err))
		} else {
			all = append(all, items...)
		}
	}
	return all, nil
}

func (a *Aggregator) FetchHighImportance(ctx context.Context) ([]core_model.RawNews, error) {
	return a.FetchAll(ctx, "A")
}
