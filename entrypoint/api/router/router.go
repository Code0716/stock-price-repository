package router

import (
	"net/http"

	"github.com/Code0716/stock-price-repository/entrypoint/api/handler"
)

func NewRouter(stockPriceHandler *handler.StockPriceHandler, stockBrandHandler *handler.StockBrandHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/daily-prices", stockPriceHandler.GetDailyPrices)
	mux.HandleFunc("/stock-brands", stockBrandHandler.GetStockBrands)
	return mux
}
