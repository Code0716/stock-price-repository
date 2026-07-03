//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/repositories"
)

// quizUniverseWindowDays 出来高/値幅の平均を取る対象営業日数。
const quizUniverseWindowDays = 20

type createQuizDailyUniverseInteractorImpl struct {
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository
	quizDailyUniverseRepository          repositories.QuizDailyUniverseRepository
}

type CreateQuizDailyUniverseInteractor interface {
	// CreateQuizDailyUniverse 直近営業日を基準に出来高がありよく動く300銘柄を選定し、当日分の出題ユニバースを作成する。
	// 既に作成済み・データ不足の場合は何もせず正常終了する（冪等）。
	CreateQuizDailyUniverse(ctx context.Context, now time.Time) error
}

func NewCreateQuizDailyUniverseInteractor(
	stockBrandsDailyStockPriceRepository repositories.StockBrandsDailyPriceRepository,
	quizDailyUniverseRepository repositories.QuizDailyUniverseRepository,
) CreateQuizDailyUniverseInteractor {
	return &createQuizDailyUniverseInteractorImpl{
		stockBrandsDailyStockPriceRepository: stockBrandsDailyStockPriceRepository,
		quizDailyUniverseRepository:          quizDailyUniverseRepository,
	}
}

func (ci *createQuizDailyUniverseInteractorImpl) CreateQuizDailyUniverse(ctx context.Context, now time.Time) error {
	dates, err := ci.stockBrandsDailyStockPriceRepository.ListRecentTradingDates(ctx, now, quizUniverseWindowDays)
	if err != nil {
		return errors.Wrap(err, "ListRecentTradingDates error")
	}
	if len(dates) < quizUniverseWindowDays {
		// データ蓄積が20営業日に満たない間は出題しない。
		return nil
	}

	quizDate := dates[0]
	from := dates[len(dates)-1]

	exists, err := ci.quizDailyUniverseRepository.ExistsByQuizDate(ctx, quizDate)
	if err != nil {
		return errors.Wrap(err, "ExistsByQuizDate error")
	}
	if exists {
		return nil
	}

	prices, err := ci.stockBrandsDailyStockPriceRepository.ListPricesByDateRange(ctx, from, quizDate)
	if err != nil {
		return errors.Wrap(err, "ListPricesByDateRange error")
	}

	entries := domain_service.SelectQuizUniverse(prices, domain_service.QuizUniverseValueTopN, domain_service.QuizUniverseRangeTopN)
	if len(entries) == 0 {
		return nil
	}

	if err := ci.quizDailyUniverseRepository.BulkCreate(ctx, entries); err != nil {
		return errors.Wrap(err, "BulkCreate error")
	}
	return nil
}
