//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

type stockBrandInteractorImpl struct {
	tx                                        repositories.Transaction
	stockBrandRepository                      repositories.StockBrandRepository
	stockBrandsDailyPriceRepository           repositories.StockBrandsDailyPriceRepository
	analyzeStockBrandPriceHistoryRepository   repositories.AnalyzeStockBrandPriceHistoryRepository
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository
	finAnnouncementRepository                 repositories.FinAnnouncementRepository
	finStatementRepository                    repositories.FinStatementRepository
	stockAPIClient                            gateway.StockAPIClient
	redisClient                               *redis.Client
}

type StockBrandInteractor interface {
	UpdateStockBrands(ctx context.Context, t time.Time) error
	GetStockBrands(ctx context.Context, keyword string, symbolFrom string, limit int, onlyMainMarkets bool) (*models.PaginatedStockBrands, error)
	GetAnalyzeStockBrandPriceHistories(ctx context.Context, filter *models.AnalyzeStockBrandPriceHistoryFilter) (*models.PaginatedAnalyzeStockBrandPriceHistories, error)
	GetMultipleSignalStocks(ctx context.Context, filter *models.MultipleSignalStockFilter) (*models.PaginatedMultipleSignalStocks, error)
	SyncFinAnnouncements(ctx context.Context) error
	GetFinAnnouncements(ctx context.Context, filter *models.FinAnnouncementFilter) (*models.PaginatedFinAnnouncements, error)
	GetNextFinAnnouncement(ctx context.Context, tickerSymbol string) (*models.FinAnnouncement, error)
	SyncFinStatements(ctx context.Context, tickerSymbol string) error
	GetFinStatements(ctx context.Context, filter *models.FinStatementFilter) ([]*models.FinStatement, error)
	// SyncFinStatementsAllStocks 全主要市場銘柄の財務情報を逐次取得・保存する。
	// intervalMs: 各APIリクエスト前のウェイト(ミリ秒, 0=ウェイトなし)。max: 処理上限(0=全件)。
	SyncFinStatementsAllStocks(ctx context.Context, intervalMs, max int) error
}

func NewStockBrandInteractor(
	tx repositories.Transaction,
	stockBrandRepository repositories.StockBrandRepository,
	stockBrandsDailyPriceRepository repositories.StockBrandsDailyPriceRepository,
	analyzeStockBrandPriceHistoryRepository repositories.AnalyzeStockBrandPriceHistoryRepository,
	stockBrandsDailyPriceForAnalyzeRepository repositories.StockBrandsDailyPriceForAnalyzeRepository,
	finAnnouncementRepository repositories.FinAnnouncementRepository,
	finStatementRepository repositories.FinStatementRepository,
	stockAPIClient gateway.StockAPIClient,
	redisClient *redis.Client,
) StockBrandInteractor {
	return &stockBrandInteractorImpl{
		tx:                                        tx,
		stockBrandRepository:                      stockBrandRepository,
		stockBrandsDailyPriceRepository:           stockBrandsDailyPriceRepository,
		analyzeStockBrandPriceHistoryRepository:   analyzeStockBrandPriceHistoryRepository,
		stockBrandsDailyPriceForAnalyzeRepository: stockBrandsDailyPriceForAnalyzeRepository,
		finAnnouncementRepository:                 finAnnouncementRepository,
		finStatementRepository:                    finStatementRepository,
		stockAPIClient:                            stockAPIClient,
		redisClient:                               redisClient,
	}
}
