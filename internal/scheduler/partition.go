package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type partitionSpec struct {
	name      string
	startDate string
	endDate   string
}

type PartitionManager struct {
	db     *gorm.DB
	table  string
	logger *log.Logger
}

func NewPartitionManager(db *gorm.DB, table string) *PartitionManager {
	return &PartitionManager{
		db:     db,
		table:  table,
		logger: log.Default(),
	}
}

func partitionSpecForDate(table string, target time.Time) partitionSpec {
	start := time.Date(target.Year(), target.Month(), 1, 0, 0, 0, 0, target.Location())
	end := start.AddDate(0, 1, 0)
	return partitionSpec{
		name:      fmt.Sprintf("%s_%04d_%02d", table, start.Year(), int(start.Month())),
		startDate: start.Format("2006-01-02"),
		endDate:   end.Format("2006-01-02"),
	}
}

func (p *PartitionManager) ensurePartition(ctx context.Context, spec partitionSpec) error {
	err := p.db.WithContext(ctx).Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s')
	`, spec.name, p.table, spec.startDate, spec.endDate)).Error
	if err != nil {
		return fmt.Errorf("create partition %s: %w", spec.name, err)
	}
	p.logger.Printf("partition %s created", spec.name)
	return nil
}

func (p *PartitionManager) EnsureMonthPartition(ctx context.Context, target time.Time) error {
	return p.ensurePartition(ctx, partitionSpecForDate(p.table, target))
}

func (p *PartitionManager) EnsureNextMonthPartition(ctx context.Context) error {
	now := time.Now()
	nextMonth := now.AddDate(0, 1, 1)
	return p.EnsureMonthPartition(ctx, nextMonth)
}

func (p *PartitionManager) CreatePartitionForYear(ctx context.Context, year int) error {
	for month := 1; month <= 12; month++ {
		target := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		if err := p.EnsureMonthPartition(ctx, target); err != nil {
			return err
		}
	}
	return nil
}

func (p *PartitionManager) ListPartitions(ctx context.Context) ([]string, error) {
	var partitions []string
	err := p.db.WithContext(ctx).Table("pg_class").
		Select("relname").
		Where("relkind = ?", "r").
		Where("relname LIKE ?", p.table+"_%").
		Order("relname").
		Pluck("relname", &partitions).Error
	if err != nil {
		return nil, fmt.Errorf("list partitions: %w", err)
	}
	return partitions, nil
}
