//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

// Sector33AverageDailyPriceRepository セクター33業種平均日足の読み取りインターフェース
type Sector33AverageDailyPriceRepository interface {
	// ListRangeAll 指定期間のセクター33業種平均日足を全業種・date 昇順で取得する。
	// sector_33_code が NULL の行は除外される。
	ListRangeAll(ctx context.Context, from, to time.Time) ([]*models.Sector33AverageDailyPrice, error)
}
