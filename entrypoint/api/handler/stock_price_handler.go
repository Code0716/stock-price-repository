package handler

import (
	"net/http"
	"regexp"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
	"go.uber.org/zap"
)

var symbolRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// getDailyPricesParams GetDailyPricesのリクエストパラメータ
type getDailyPricesParams struct {
	symbol    string
	from      *time.Time
	to        *time.Time
	sortOrder *models.SortOrder
}

type StockPriceHandler struct {
	usecase    usecase.StockBrandsDailyPriceInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewStockPriceHandler(u usecase.StockBrandsDailyPriceInteractor, h driver.HTTPServer, l *zap.Logger) *StockPriceHandler {
	return &StockPriceHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

// validateGetDailyPricesParams GetDailyPricesのリクエストパラメータをバリデーションする
func (h *StockPriceHandler) validateGetDailyPricesParams(r *http.Request) (*getDailyPricesParams, error) {
	params := &getDailyPricesParams{}

	// symbol パラメータの取得とバリデーション
	params.symbol = h.httpServer.GetQueryParam(r, "symbol")
	if params.symbol == "" {
		return nil, &validationError{message: "シンボルは必須です"}
	}

	if len(params.symbol) > 10 {
		return nil, &validationError{message: "シンボルが長すぎます"}
	}

	if !symbolRegex.MatchString(params.symbol) {
		return nil, &validationError{message: "シンボルは英数字である必要があります"}
	}

	// from パラメータの取得とバリデーション
	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "fromの日付形式が不正です"}
	}
	params.from = from

	// to パラメータの取得とバリデーション
	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "toの日付形式が不正です"}
	}
	params.to = to

	// order パラメータの取得とバリデーション
	orderParam := h.httpServer.GetQueryParam(r, "order")
	if orderParam != "" {
		if orderParam != string(models.SortOrderAsc) && orderParam != string(models.SortOrderDesc) {
			return nil, &validationError{message: "orderはascまたはdescである必要があります"}
		}
		order := models.SortOrder(orderParam)
		params.sortOrder = &order
	}

	return params, nil
}

func (h *StockPriceHandler) GetDailyPrices(w http.ResponseWriter, r *http.Request) {
	// パラメータのバリデーション
	params, err := h.validateGetDailyPricesParams(r)
	if err != nil {
		if verr, ok := err.(*validationError); ok {
			http.Error(w, verr.message, http.StatusBadRequest)
			return
		}
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	// ユースケース呼び出し
	prices, err := h.usecase.GetDailyStockPricesWithOrder(r.Context(), params.symbol, params.from, params.to, params.sortOrder)
	if err != nil {
		h.logger.Error("failed to get daily stock prices", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	respondJSON(w, h.logger, prices)
}
