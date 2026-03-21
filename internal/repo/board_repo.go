package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"stock/internal/model"
)

type BoardRepo struct {
	db *gorm.DB
}

func NewBoardRepo(db *gorm.DB) *BoardRepo {
	return &BoardRepo{db: db}
}

func (r *BoardRepo) GetAll(ctx context.Context) ([]model.BoardRule, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT board_code, board_name, limit_up_ratio, limit_down_ratio, code_patterns
		FROM boards ORDER BY board_code`).Rows()
	if err != nil {
		return nil, fmt.Errorf("query boards: %w", err)
	}
	defer rows.Close()

	var results []model.BoardRule
	for rows.Next() {
		var br model.BoardRule
		var codeStr string
		if err := rows.Scan(&br.BoardCode, &br.BoardName, &br.LimitUpRatio, &br.LimitDownRatio, &codeStr); err != nil {
			return nil, fmt.Errorf("scan board row: %w", err)
		}
		if codeStr != "" {
			if err := json.Unmarshal([]byte(codeStr), &br.CodePatterns); err != nil {
				return nil, fmt.Errorf("unmarshal code_patterns: %w", err)
			}
		}
		results = append(results, br)
	}
	return results, rows.Err()
}

func (r *BoardRepo) GetByCode(ctx context.Context, boardCode string) (*model.BoardRule, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT board_code, board_name, limit_up_ratio, limit_down_ratio, code_patterns
		FROM boards WHERE board_code = ?`, boardCode).Rows()
	if err != nil {
		return nil, fmt.Errorf("query board: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var br model.BoardRule
	var codeStr string
	if err := rows.Scan(&br.BoardCode, &br.BoardName, &br.LimitUpRatio, &br.LimitDownRatio, &codeStr); err != nil {
		return nil, fmt.Errorf("scan board row: %w", err)
	}
	if codeStr != "" {
		if err := json.Unmarshal([]byte(codeStr), &br.CodePatterns); err != nil {
			return nil, fmt.Errorf("unmarshal code_patterns: %w", err)
		}
	}
	return &br, rows.Err()
}

func (r *BoardRepo) Upsert(ctx context.Context, rule *model.BoardRule) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "board_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"board_name", "limit_up_ratio", "limit_down_ratio", "code_patterns",
		}),
	}).Create(rule).Error
}

func (r *BoardRepo) InitDefaultBoards(ctx context.Context) error {
	defaultBoards := []model.BoardRule{
		{BoardCode: model.BoardMain, BoardName: "主板", LimitUpRatio: 0.10, LimitDownRatio: -0.10, CodePatterns: []string{"60____", "000___", "001___", "002___"}},
		{BoardCode: model.BoardGEM, BoardName: "创业板", LimitUpRatio: 0.20, LimitDownRatio: -0.20, CodePatterns: []string{"300___", "301___"}},
		{BoardCode: model.BoardSTAR, BoardName: "科创板", LimitUpRatio: 0.20, LimitDownRatio: -0.20, CodePatterns: []string{"688___"}},
		{BoardCode: model.BoardBSE, BoardName: "北交所", LimitUpRatio: 0.30, LimitDownRatio: -0.30, CodePatterns: []string{"8____", "4____"}},
	}
	for _, b := range defaultBoards {
		if err := r.Upsert(ctx, &b); err != nil {
			return fmt.Errorf("init board %s: %w", b.BoardCode, err)
		}
	}
	return nil
}
