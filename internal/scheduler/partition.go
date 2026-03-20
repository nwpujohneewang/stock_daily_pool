package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

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

func (p *PartitionManager) EnsureNextMonthPartition(ctx context.Context) error {
	now := time.Now()
	nextMonth := now.AddDate(0, 1, 1)
	year := nextMonth.Year()
	month := int(nextMonth.Month())

	partitionName := fmt.Sprintf("%s_%04d_%02d", p.table, year, month)
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	endDate := fmt.Sprintf("%04d-%02d-01", year, month+1)

	if month == 12 {
		endDate = fmt.Sprintf("%04d-01-01", year+1)
	}

	err := p.db.WithContext(ctx).Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, p.table, startDate, endDate)).Error

	if err != nil {
		return fmt.Errorf("create partition %s: %w", partitionName, err)
	}

	p.logger.Printf("partition %s created", partitionName)
	return nil
}

func (p *PartitionManager) CreatePartitionForYear(ctx context.Context, year int) error {
	for month := 1; month <= 12; month++ {
		partitionName := fmt.Sprintf("%s_%04d_%02d", p.table, year, month)
		startDate := fmt.Sprintf("%04d-%02d-01", year, month)
		var endDate string
		if month == 12 {
			endDate = fmt.Sprintf("%04d-01-01", year+1)
		} else {
			endDate = fmt.Sprintf("%04d-%02d-01", year, month+1)
		}

		err := p.db.WithContext(ctx).Exec(fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s
			PARTITION OF %s
			FOR VALUES FROM ('%s') TO ('%s')
		`, partitionName, p.table, startDate, endDate)).Error

		if err != nil {
			return fmt.Errorf("create partition %s: %w", partitionName, err)
		}
	}
	return nil
}

func (p *PartitionManager) ListPartitions(ctx context.Context) ([]string, error) {
	rows, err := p.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT relname FROM pg_class
		WHERE relkind = 'r'
		AND relname LIKE '%s_%%'
		ORDER BY relname
	`, p.table)).Rows()
	if err != nil {
		return nil, fmt.Errorf("list partitions: %w", err)
	}
	defer rows.Close()

	var partitions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan partition: %w", err)
		}
		partitions = append(partitions, name)
	}
	return partitions, nil
}
