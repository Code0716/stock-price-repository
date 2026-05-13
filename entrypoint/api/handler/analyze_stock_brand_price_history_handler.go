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

type AnalyzeHistoryPaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type GetAnalyzeStockBrandPriceHistoriesResponse struct {
	Histories  []*models.AnalyzeStockBrandPriceHistory `json:"histories"`
	Pagination *AnalyzeHistoryPaginationInfo            `json:"pagination"`
}

type getAnalyzeStockBrandPriceHistoriesParams struct {
	symbol string
	action string
	method string
	sortBy string
	order  string
	page   int
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
		limit:  defaultAnalyzeStockBrandPriceHistoryLimit,
		page:   1,
		sortBy: models.AnalyzeStockBrandPriceHistorySortByCreatedAt,
		order:  models.AnalyzeStockBrandPriceHistoryOrderDesc,
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

	sortBy := h.httpServer.GetQueryParam(r, "sort_by")
	if sortBy != "" {
		switch sortBy {
		case models.AnalyzeStockBrandPriceHistorySortByCreatedAt,
			models.AnalyzeStockBrandPriceHistorySortByProfit,
			models.AnalyzeStockBrandPriceHistorySortByProfitRate:
			params.sortBy = sortBy
		default:
			return nil, &validationError{message: "sort_byはcreated_at, profit, profit_rateのいずれかである必要があります"}
		}
	}

	order := h.httpServer.GetQueryParam(r, "order")
	if order != "" {
		switch order {
		case models.AnalyzeStockBrandPriceHistoryOrderAsc, models.AnalyzeStockBrandPriceHistoryOrderDesc:
			params.order = order
		default:
			return nil, &validationError{message: "orderはascまたはdescである必要があります"}
		}
	}

	pageStr := h.httpServer.GetQueryParam(r, "page")
	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil, &validationError{message: "pageは数値である必要があります"}
		}
		if page <= 0 {
			return nil, &validationError{message: "pageは正の整数である必要があります"}
		}
		params.page = page
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
		SortBy:       params.sortBy,
		Order:        params.order,
		Page:         params.page,
		Limit:        params.limit,
	})
	if err != nil {
		h.logger.Error("failed to get analyze stock brand price histories", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	respondJSON(w, h.logger, &GetAnalyzeStockBrandPriceHistoriesResponse{
		Histories: result.Histories,
		Pagination: &AnalyzeHistoryPaginationInfo{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}
