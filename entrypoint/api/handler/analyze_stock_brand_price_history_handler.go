package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

// validateAnalyzeHistoryAction は action パラメータを検証する。
func validateAnalyzeHistoryAction(action string) error {
	if action == "" ||
		action == models.AnalyzeStockBrandPriceHistoryActionBuy ||
		action == models.AnalyzeStockBrandPriceHistoryActionSell {
		return nil
	}
	return &validationError{message: "actionはBuyまたはSellである必要があります"}
}

// parseBoundedInt はクエリパラメータを正の整数として読み取る。
// max に 0 を渡すと上限チェックを省略する。値が空文字列の場合は defaultVal を返す。
func parseBoundedInt(server driver.HTTPServer, r *http.Request, key string, defaultVal, max int) (int, error) {
	str := server.GetQueryParam(r, key)
	if str == "" {
		return defaultVal, nil
	}
	v, err := strconv.Atoi(str)
	if err != nil {
		return 0, &validationError{message: fmt.Sprintf("%sは数値である必要があります", key)}
	}
	if v <= 0 {
		return 0, &validationError{message: fmt.Sprintf("%sは正の整数である必要があります", key)}
	}
	if max > 0 && v > max {
		return 0, &validationError{message: fmt.Sprintf("%sは%d以下である必要があります", key, max)}
	}
	return v, nil
}

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

type AnalyzeHistoryDatePaginationInfo struct {
	DatePage       int   `json:"date_page"`
	DateLimit      int   `json:"date_limit"`
	TotalDates     int64 `json:"total_dates"`
	TotalDatePages int   `json:"total_date_pages"`
}

type GetAnalyzeStockBrandPriceHistoriesResponse struct {
	Histories      []*models.AnalyzeStockBrandPriceHistory `json:"histories"`
	Pagination     *AnalyzeHistoryPaginationInfo            `json:"pagination"`
	DatePagination *AnalyzeHistoryDatePaginationInfo        `json:"date_pagination,omitempty"`
}

type getAnalyzeStockBrandPriceHistoriesParams struct {
	symbol    string
	action    string
	method    string
	sortBy    string
	order     string
	page      int
	limit     int
	datePage  int
	dateLimit int
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
	if err := validateAnalyzeHistoryAction(params.action); err != nil {
		return nil, err
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

	page, err := parseBoundedInt(h.httpServer, r, "page", 1, 0)
	if err != nil {
		return nil, err
	}
	params.page = page

	limit, err := parseBoundedInt(h.httpServer, r, "limit", defaultAnalyzeStockBrandPriceHistoryLimit, maxAnalyzeStockBrandPriceHistoryLimit)
	if err != nil {
		return nil, err
	}
	params.limit = limit

	datePage, err := parseBoundedInt(h.httpServer, r, "date_page", 0, 0)
	if err != nil {
		return nil, err
	}
	params.datePage = datePage

	dateLimit, err := parseBoundedInt(h.httpServer, r, "date_limit", 0, 50)
	if err != nil {
		return nil, err
	}
	params.dateLimit = dateLimit

	return params, nil
}

func (h *AnalyzeStockBrandPriceHistoryHandler) GetAnalyzeStockBrandPriceHistories(w http.ResponseWriter, r *http.Request) {
	params, err := h.validateGetAnalyzeStockBrandPriceHistoriesParams(r)
	if err != nil {
		writeError(w, h.logger, "failed to validate get analyze stock brand price histories params", err)
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
		DatePage:     params.datePage,
		DateLimit:    params.dateLimit,
	})
	if err != nil {
		writeError(w, h.logger, "failed to get analyze stock brand price histories", err)
		return
	}

	resp := &GetAnalyzeStockBrandPriceHistoriesResponse{
		Histories: result.Histories,
		Pagination: &AnalyzeHistoryPaginationInfo{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	}
	if params.datePage > 0 {
		resp.DatePagination = &AnalyzeHistoryDatePaginationInfo{
			DatePage:       result.DatePage,
			DateLimit:      result.DateLimit,
			TotalDates:     result.TotalDates,
			TotalDatePages: result.TotalDatePages,
		}
	}
	respondJSON(w, h.logger, resp)
}
