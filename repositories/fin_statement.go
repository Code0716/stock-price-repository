//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type FinStatementRepository interface {
	// Upsert 財務情報を登録または更新する
	Upsert(ctx context.Context, statements []*models.FinStatement) error
	// FindBySymbol 銘柄の財務情報を新しい順に取得する
	FindBySymbol(ctx context.Context, filter *models.FinStatementFilter) ([]*models.FinStatement, error)
}
