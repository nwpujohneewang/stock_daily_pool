package scheduler

import (
	"context"
	"log"
	"time"

	"stock/internal/service"
)

type Ticker struct {
	cron           *cronWrapper
	monitorService *service.MonitorService
	logger         *log.Logger
	intervalSec    int
}

type cronWrapper struct{}

func (c *cronWrapper) Now() time.Time {
	return time.Now()
}

func NewTicker(
	intervalSec int,
	monitorService *service.MonitorService,
) *Ticker {
	return &Ticker{
		monitorService: monitorService,
		logger:         log.Default(),
		intervalSec:    intervalSec,
	}
}

func (t *Ticker) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(t.intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !t.isTradingTime() {
				continue
			}
			if err := t.monitorService.ProcessTick(ctx, time.Now().Format("2006-01-02")); err != nil {
				t.logger.Printf("process tick failed: %v", err)
			}
		}
	}
}

func (t *Ticker) isTradingTime() bool {
	now := time.Now()
	hour := now.Hour()
	min := now.Minute()

	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}

	if hour < 9 || hour >= 15 {
		return false
	}

	if hour == 9 && min < 25 {
		return false
	}

	return true
}

func (t *Ticker) Stop() error {
	return nil
}
