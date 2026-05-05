package handler

import (
	"net/http"
	"strconv"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

const (
	defaultAnalyzeStockBrandPriceHistoryLimit = 100
	maxAnalyzeStockBrandPriceHistoryLimit     = 1000
)

type GetAnalyzeStockBrandPriceHistoriesResponse struct {
	Histories  []*models.AnalyzeStockBrandPriceHistory `json:"histories"`
	Pagination *PaginationInfo                         `json:"pagination,omitempty"`
}

type getAnalyzeStockBrandPriceHistoriesParams struct {
	symbol string
	action string
	method string
	cursor string
	limit  int
}

type AnalyzeStockBrandPriceHistoryHandler struct {
	usecase    usecase.StockBrandInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewAnalyzeStockBrandPriceHistoryHandler(u usecase.StockBrandInteractor, h driver.HTTPServer, l *zap.Logger) *AnalyzeStockBrandPriceHistoryHandler {
	return &AnalyzeStockBrandPriceHistoryHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

func (h *AnalyzeStockBrandPriceHistoryHandler) validateGetAnalyzeStockBrandPriceHistoriesParams(r *http.Request) (*getAnalyzeStockBrandPriceHistoriesParams, error) {
	params := &getAnalyzeStockBrandPriceHistoriesParams{
		limit: defaultAnalyzeStockBrandPriceHistoryLimit,
	}

	params.symbol = h.httpServer.GetQueryParam(r, "symbol")
	if params.symbol != "" {
		if len(params.symbol) > 10 {
			return nil, &validationError{message: "symbolが長すぎます"}
		}
		if !alphanumericOptionalRegex.MatchString(params.symbol) {
			return nil, &validationError{message: "symbolは英数字である必要があります"}
		}
	}

	params.action = h.httpServer.GetQueryParam(r, "action")
	if params.action != "" &&
		params.action != models.AnalyzeStockBrandPriceHistoryActionBuy &&
		params.action != models.AnalyzeStockBrandPriceHistoryActionSell {
		return nil, &validationError{message: "actionはBuyまたはSellである必要があります"}
	}

	params.method = h.httpServer.GetQueryParam(r, "method")
	if len(params.method) > 255 {
		return nil, &validationError{message: "methodが長すぎます"}
	}

	params.cursor = h.httpServer.GetQueryParam(r, "cursor")
	if len(params.cursor) > 36 {
		return nil, &validationError{message: "cursorが長すぎます"}
	}

	limitStr := h.httpServer.GetQueryParam(r, "limit")
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, &validationError{message: "limitは数値である必要があります"}
		}
		if limit <= 0 {
			return nil, &validationError{message: "limitは正の整数である必要があります"}
		}
		if limit > maxAnalyzeStockBrandPriceHistoryLimit {
			return nil, &validationError{message: "limitは1000以下である必要があります"}
		}
		params.limit = limit
	}

	return params, nil
}

func (h *AnalyzeStockBrandPriceHistoryHandler) GetAnalyzeStockBrandPriceHistories(w http.ResponseWriter, r *http.Request) {
	params, err := h.validateGetAnalyzeStockBrandPriceHistoriesParams(r)
	if err != nil {
		if verr, ok := err.(*validationError); ok {
			http.Error(w, verr.message, http.StatusBadRequest)
			return
		}
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	result, err := h.usecase.GetAnalyzeStockBrandPriceHistories(r.Context(), &models.AnalyzeStockBrandPriceHistoryFilter{
		TickerSymbol: params.symbol,
		Action:       params.action,
		Method:       params.method,
		Cursor:       params.cursor,
		Limit:        params.limit,
	})
	if err != nil {
		h.logger.Error("failed to get analyze stock brand price histories", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	respondJSON(w, h.logger, &GetAnalyzeStockBrandPriceHistoriesResponse{
		Histories: result.Histories,
		Pagination: &PaginationInfo{
			NextCursor: result.NextCursor,
			Limit:      result.Limit,
		},
	})
}
