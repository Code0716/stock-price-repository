//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

package repositories

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/models"
)

type MarketCalendarRepository interface {
	// UpsertDays 取引カレンダーを日付キーで upsert する。
	UpsertDays(ctx context.Context, days []*models.MarketCalendarDay) error
	// ListDays from〜to（両端含む）の取引カレンダーを日付昇順で取得する。
	ListDays(ctx context.Context, from, to time.Time) ([]*models.MarketCalendarDay, error)
	// ReplaceEvents 指定 kind の from〜to（両端含む）のイベントを削除してから events を作成する（洗い替え）。
	// 削除と作成は同一トランザクションで呼ぶこと。
	ReplaceEvents(ctx context.Context, kind string, from, to time.Time, events []*models.MarketEvent) error
	// ListEvents from〜to（両端含む）のイベントを日付昇順で取得する。
	ListEvents(ctx context.Context, from, to time.Time) ([]*models.MarketEvent, error)
}
