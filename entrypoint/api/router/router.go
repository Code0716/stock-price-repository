package router

import (
	"net/http"

	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
)

func NewRouter(
	stockPriceHandler *handler.StockPriceHandler,
	stockBrandHandler *handler.StockBrandHandler,
	analyzeStockBrandPriceHistoryHandler *handler.AnalyzeStockBrandPriceHistoryHandler,
	multipleSignalStocksHandler *handler.MultipleSignalStocksHandler,
	finAnnouncementHandler *handler.FinAnnouncementHandler,
	finStatementHandler *handler.FinStatementHandler,
) *http.ServeMux {
	mux := http.NewServeMux()
	if stockPriceHandler != nil {
		mux.HandleFunc("/daily-prices", stockPriceHandler.GetDailyPrices)
	}
	if stockBrandHandler != nil {
		mux.HandleFunc("/stock-brands", stockBrandHandler.GetStockBrands)
	}
	if analyzeStockBrandPriceHistoryHandler != nil {
		mux.HandleFunc("/analyze-stock-brand-price-histories", analyzeStockBrandPriceHistoryHandler.GetAnalyzeStockBrandPriceHistories)
	}
	if multipleSignalStocksHandler != nil {
		mux.HandleFunc("/multiple-signal-stocks", multipleSignalStocksHandler.GetMultipleSignalStocks)
	}
	if finAnnouncementHandler != nil {
		mux.HandleFunc("/fin-announcements", finAnnouncementHandler.GetFinAnnouncements)
		mux.HandleFunc("/fin-announcements/next", finAnnouncementHandler.GetNextFinAnnouncement)
	}
	if finStatementHandler != nil {
		mux.HandleFunc("/fin-statements", finStatementHandler.GetFinStatements)
	}
	return mux
}
