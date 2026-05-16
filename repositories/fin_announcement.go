//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package repositories

import (
	"context"

	"github.com/Code0716/stock-price-repository/models"
)

type FinAnnouncementRepository interface {
	// Upsert 決算発表予定を登録または更新する
	Upsert(ctx context.Context, announcements []*models.FinAnnouncement) error
	// FindWithFilter 条件に一致する決算発表予定を取得する
	FindWithFilter(ctx context.Context, filter *models.FinAnnouncementFilter) ([]*models.FinAnnouncement, error)
	// CountWithFilter 条件に一致する決算発表予定の総件数を取得する
	CountWithFilter(ctx context.Context, filter *models.FinAnnouncementFilter) (int64, error)
	// FindNextBySymbol 銘柄の次回決算発表予定日を取得する（最も直近の未来日）
	FindNextBySymbol(ctx context.Context, tickerSymbol string) (*models.FinAnnouncement, error)
}
