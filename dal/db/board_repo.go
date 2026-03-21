package db

import (
	"context"
	"fmt"
	"stock/model/dal_model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ BoardRepository = (*BoardRepoImpl)(nil)

type BoardRepository interface {
	GetAll(ctx context.Context) ([]dal_model.BoardRule, error)
	GetByCode(ctx context.Context, boardCode string) (*dal_model.BoardRule, error)
	Upsert(ctx context.Context, rule *dal_model.BoardRule) error
	InitDefaultBoards(ctx context.Context) error
}

type BoardRepoImpl struct{}

func NewBoardRepository() *BoardRepoImpl {
	return &BoardRepoImpl{}
}

func (r BoardRepoImpl) GetAll(ctx context.Context) ([]dal_model.BoardRule, error) {
	var results []dal_model.BoardRule
	if err := PostgresStockDB(ctx).Find(&results).Error; err != nil {
		return nil, fmt.Errorf("query boards: %w", err)
	}
	for i := range results {
		if results[i].CodePatterns == nil {
			results[i].CodePatterns = []string{}
		}
	}
	return results, nil
}

func (r BoardRepoImpl) GetByCode(ctx context.Context, boardCode string) (*dal_model.BoardRule, error) {
	var br dal_model.BoardRule
	if err := PostgresStockDB(ctx).Where("board_code = ?", boardCode).First(&br).Error; err != nil {
		return nil, err
	}
	if br.CodePatterns == nil {
		br.CodePatterns = []string{}
	}
	return &br, nil
}

func (r BoardRepoImpl) Upsert(ctx context.Context, rule *dal_model.BoardRule) error {
	return PostgresStockDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "board_code"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"board_name":       gorm.Expr("EXCLUDED.board_name"),
			"limit_up_ratio":   gorm.Expr("EXCLUDED.limit_up_ratio"),
			"limit_down_ratio": gorm.Expr("EXCLUDED.limit_down_ratio"),
			"code_patterns":    gorm.Expr("EXCLUDED.code_patterns"),
		}),
	}).Create(rule).Error
}

func (r BoardRepoImpl) InitDefaultBoards(ctx context.Context) error {
	defaultBoards := []dal_model.BoardRule{
		{BoardCode: dal_model.BoardMain, BoardName: "主板", LimitUpRatio: 0.10, LimitDownRatio: -0.10, CodePatterns: []string{"60____", "000___", "001___", "002___"}},
		{BoardCode: dal_model.BoardGEM, BoardName: "创业板", LimitUpRatio: 0.20, LimitDownRatio: -0.20, CodePatterns: []string{"300___", "301___"}},
		{BoardCode: dal_model.BoardSTAR, BoardName: "科创板", LimitUpRatio: 0.20, LimitDownRatio: -0.20, CodePatterns: []string{"688___"}},
		{BoardCode: dal_model.BoardBSE, BoardName: "北交所", LimitUpRatio: 0.30, LimitDownRatio: -0.30, CodePatterns: []string{"8____", "4____"}},
	}
	for _, b := range defaultBoards {
		if err := r.Upsert(ctx, &b); err != nil {
			return fmt.Errorf("init board %s: %w", b.BoardCode, err)
		}
	}
	return nil
}
