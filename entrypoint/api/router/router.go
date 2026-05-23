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
	daytradeHandler *handler.DaytradeHandler,
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
	if daytradeHandler != nil {
		mux.HandleFunc("/daytrade/executions/import", daytradeHandler.ImportSBICsv)
		mux.HandleFunc("/daytrade/summary", daytradeHandler.GetSummary)
		mux.HandleFunc("/daytrade/summary-by-symbol", daytradeHandler.GetSummaryByTickerSymbol)
		mux.HandleFunc("/daytrade/executions", daytradeHandler.GetExecutionsByDate)
		mux.HandleFunc("/daytrade/range", daytradeHandler.GetCoveredRange)
		mux.HandleFunc("/daytrade/stats", daytradeHandler.GetStats)
	}
	return mux
}
