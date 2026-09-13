package handler

import (
	"net/http"
	"strconv"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

// validationError バリデーションエラー
type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

// GetStockBrandsResponse 銘柄一覧取得APIのレスポンス
type GetStockBrandsResponse struct {
	StockBrands []*models.StockBrand `json:"stock_brands"`
	Pagination  *PaginationInfo      `json:"pagination,omitempty"`
}

// PaginationInfo ページネーション情報
type PaginationInfo struct {
	NextCursor *string `json:"next_cursor"`
	Limit      int     `json:"limit,omitempty"`
}

// getStockBrandsParams GetStockBrandsのリクエストパラメータ
type getStockBrandsParams struct {
	keyword         string
	symbolFrom      string
	limit           int
	onlyMainMarkets bool
}

type StockBrandHandler struct {
	usecase    usecase.StockBrandInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewStockBrandHandler(u usecase.StockBrandInteractor, h driver.HTTPServer, l *zap.Logger) *StockBrandHandler {
	return &StockBrandHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

// validateGetStockBrandsParams GetStockBrandsのリクエストパラメータをバリデーションする
func (h *StockBrandHandler) validateGetStockBrandsParams(r *http.Request) (*getStockBrandsParams, error) {
	params := &getStockBrandsParams{}

	// keyword パラメータの取得とバリデーション（銘柄コード前方一致 OR 銘柄名部分一致）
	params.keyword = h.httpServer.GetQueryParam(r, "keyword")
	if params.keyword != "" {
		if len([]rune(params.keyword)) > 50 {
			return nil, &validationError{message: "keywordが長すぎます"}
		}
	}

	// symbolFrom パラメータの取得とバリデーション
	params.symbolFrom = h.httpServer.GetQueryParam(r, "symbol_from")
	if params.symbolFrom != "" {
		if len(params.symbolFrom) > 10 {
			return nil, &validationError{message: "symbol_fromが長すぎます"}
		}
		if !alphanumericOptionalRegex.MatchString(params.symbolFrom) {
			return nil, &validationError{message: "symbol_fromは英数字である必要があります"}
		}
	}

	// limit パラメータの取得とバリデーション
	limitStr := h.httpServer.GetQueryParam(r, "limit")
	if limitStr != "" {
		var err error
		params.limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, &validationError{message: "limitは数値である必要があります"}
		}
		if params.limit <= 0 {
			return nil, &validationError{message: "limitは正の整数である必要があります"}
		}
		if params.limit > 10000 {
			return nil, &validationError{message: "limitは10000以下である必要があります"}
		}
	}

	// only_main_markets パラメータの取得とバリデーション
	onlyMainMarketsStr := h.httpServer.GetQueryParam(r, "only_main_markets")
	if onlyMainMarketsStr != "" {
		var err error
		params.onlyMainMarkets, err = strconv.ParseBool(onlyMainMarketsStr)
		if err != nil {
			return nil, &validationError{message: "only_main_marketsはtrue/falseである必要があります"}
		}
	}

	return params, nil
}

// buildGetStockBrandsResponse GetStockBrandsのレスポンスを構築する
func buildGetStockBrandsResponse(result *models.PaginatedStockBrands, limit int) *GetStockBrandsResponse {
	response := &GetStockBrandsResponse{
		StockBrands: result.Brands,
	}

	// ページネーション情報の設定
	if limit > 0 {
		response.Pagination = &PaginationInfo{
			NextCursor: result.NextCursor,
			Limit:      result.Limit,
		}
	}

	return response
}

func (h *StockBrandHandler) GetStockBrands(w http.ResponseWriter, r *http.Request) {
	// パラメータのバリデーション
	params, err := h.validateGetStockBrandsParams(r)
	if err != nil {
		writeError(w, h.logger, "failed to validate get stock brands params", err)
		return
	}

	// ユースケース呼び出し
	result, err := h.usecase.GetStockBrands(r.Context(), params.keyword, params.symbolFrom, params.limit, params.onlyMainMarkets)
	if err != nil {
		writeError(w, h.logger, "failed to get stock brands", err)
		return
	}

	// レスポンス構築
	response := buildGetStockBrandsResponse(result, params.limit)

	// JSON レスポンス
	respondJSON(w, h.logger, response)
}
