package router

import (
	"net/http"

	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
)

func NewRouter(stockPriceHandler *handler.StockPriceHandler, stockBrandHandler *handler.StockBrandHandler) *http.ServeMux {
	mux := http.NewServeMux()
	if stockPriceHandler != nil {
		mux.HandleFunc("/daily-prices", stockPriceHandler.GetDailyPrices)
	}
	if stockBrandHandler != nil {
		mux.HandleFunc("/stock-brands", stockBrandHandler.GetStockBrands)
	}
	return mux
}
