package usecase

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/models"
)

// GetFinStatements 銘柄の財務情報を新しい順に取得する
func (si *stockBrandInteractorImpl) GetFinStatements(ctx context.Context, filter *models.FinStatementFilter) ([]*models.FinStatement, error) {
	if filter == nil {
		filter = &models.FinStatementFilter{}
	}
	if filter.Limit <= 0 {
		filter.Limit = 8
	}

	statements, err := si.finStatementRepository.FindBySymbol(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "finStatementRepository.FindBySymbol error")
	}
	return statements, nil
}
