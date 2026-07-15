package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

const (
	defaultMultipleSignalStocksLimit = 100
	maxMultipleSignalStocksLimit     = 500
)

type GetMultipleSignalStocksResponse struct {
	Stocks     []*models.MultipleSignalStock `json:"stocks"`
	Pagination *PaginationInfo               `json:"pagination,omitempty"`
}

type MultipleSignalStocksHandler struct {
	usecase    usecase.StockBrandInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewMultipleSignalStocksHandler(u usecase.StockBrandInteractor, h driver.HTTPServer, l *zap.Logger) *MultipleSignalStocksHandler {
	return &MultipleSignalStocksHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

type getMultipleSignalStocksParams struct {
	date   *time.Time
	cursor string
	limit  int
}

func (h *MultipleSignalStocksHandler) validateGetMultipleSignalStocksParams(r *http.Request) (*getMultipleSignalStocksParams, error) {
	params := &getMultipleSignalStocksParams{
		limit: defaultMultipleSignalStocksLimit,
	}

	dateStr := h.httpServer.GetQueryParam(r, "date")
	if dateStr != "" {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, &validationError{message: "dateはYYYY-MM-DD形式で指定してください"}
		}
		params.date = &t
	}

	params.cursor = h.httpServer.GetQueryParam(r, "cursor")
	if len(params.cursor) > 20 {
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
		if limit > maxMultipleSignalStocksLimit {
			return nil, &validationError{message: "limitは500以下である必要があります"}
		}
		params.limit = limit
	}

	return params, nil
}

func (h *MultipleSignalStocksHandler) GetMultipleSignalStocks(w http.ResponseWriter, r *http.Request) {
	params, err := h.validateGetMultipleSignalStocksParams(r)
	if err != nil {
		writeError(w, h.logger, "failed to validate get multiple signal stocks params", err)
		return
	}

	result, err := h.usecase.GetMultipleSignalStocks(r.Context(), &models.MultipleSignalStockFilter{
		Date:   params.date,
		Cursor: params.cursor,
		Limit:  params.limit,
	})
	if err != nil {
		writeError(w, h.logger, "failed to get multiple signal stocks", err)
		return
	}

	respondJSON(w, h.logger, &GetMultipleSignalStocksResponse{
		Stocks: result.Stocks,
		Pagination: &PaginationInfo{
			NextCursor: result.NextCursor,
			Limit:      result.Limit,
		},
	})
}
