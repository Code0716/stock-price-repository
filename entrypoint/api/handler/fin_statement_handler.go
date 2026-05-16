package handler

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
)

type FinStatementHandler struct {
	usecase    usecase.StockBrandInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewFinStatementHandler(u usecase.StockBrandInteractor, h driver.HTTPServer, l *zap.Logger) *FinStatementHandler {
	return &FinStatementHandler{usecase: u, httpServer: h, logger: l}
}

type FinStatementResponse struct {
	ID                   string  `json:"id"`
	TickerSymbol         string  `json:"tickerSymbol"`
	DisclosedDate        string  `json:"disclosedDate"`
	FiscalYearEnd        *string `json:"fiscalYearEnd"`
	TypeOfDocument       string  `json:"typeOfDocument"`
	TypeOfCurrentPeriod  string  `json:"typeOfCurrentPeriod"`
	NetSales             *string `json:"netSales"`
	OperatingProfit      *string `json:"operatingProfit"`
	OrdinaryProfit       *string `json:"ordinaryProfit"`
	Profit               *string `json:"profit"`
	EarningsPerShare     *string `json:"earningsPerShare"`
	BookValuePerShare    *string `json:"bookValuePerShare"`
	ForecastNetSales     *string `json:"forecastNetSales"`
	ForecastOperatingProfit *string `json:"forecastOperatingProfit"`
	ForecastProfit       *string `json:"forecastProfit"`
	ForecastEPS          *string `json:"forecastEps"`
}

type GetFinStatementsResponse struct {
	Statements []*FinStatementResponse `json:"statements"`
}

func toFinStatementResponse(s *models.FinStatement) *FinStatementResponse {
	resp := &FinStatementResponse{
		ID:                  s.ID,
		TickerSymbol:        s.TickerSymbol,
		DisclosedDate:       s.DisclosedDate.Format("2006-01-02"),
		TypeOfDocument:      s.TypeOfDocument,
		TypeOfCurrentPeriod: s.TypeOfCurrentPeriod,
	}
	if s.FiscalYearEnd != nil {
		v := s.FiscalYearEnd.Format("2006-01-02")
		resp.FiscalYearEnd = &v
	}
	if s.NetSales != nil {
		v := s.NetSales.String()
		resp.NetSales = &v
	}
	if s.OperatingProfit != nil {
		v := s.OperatingProfit.String()
		resp.OperatingProfit = &v
	}
	if s.OrdinaryProfit != nil {
		v := s.OrdinaryProfit.String()
		resp.OrdinaryProfit = &v
	}
	if s.Profit != nil {
		v := s.Profit.String()
		resp.Profit = &v
	}
	if s.EarningsPerShare != nil {
		v := s.EarningsPerShare.String()
		resp.EarningsPerShare = &v
	}
	if s.BookValuePerShare != nil {
		v := s.BookValuePerShare.String()
		resp.BookValuePerShare = &v
	}
	if s.ForecastNetSales != nil {
		v := s.ForecastNetSales.String()
		resp.ForecastNetSales = &v
	}
	if s.ForecastOperatingProfit != nil {
		v := s.ForecastOperatingProfit.String()
		resp.ForecastOperatingProfit = &v
	}
	if s.ForecastProfit != nil {
		v := s.ForecastProfit.String()
		resp.ForecastProfit = &v
	}
	if s.ForecastEPS != nil {
		v := s.ForecastEPS.String()
		resp.ForecastEPS = &v
	}
	return resp
}

// GetFinStatements GET /fin-statements?symbol=XXXX&limit=8
func (h *FinStatementHandler) GetFinStatements(w http.ResponseWriter, r *http.Request) {
	symbol := h.httpServer.GetQueryParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "symbolは必須です", http.StatusBadRequest)
		return
	}
	if len(symbol) > 10 {
		http.Error(w, "symbolが長すぎます", http.StatusBadRequest)
		return
	}

	limit, err := parseBoundedInt(h.httpServer, r, "limit", 8, 20)
	if err != nil {
		if verr, ok := err.(*validationError); ok {
			http.Error(w, verr.message, http.StatusBadRequest)
			return
		}
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	result, err := h.usecase.GetFinStatements(r.Context(), &models.FinStatementFilter{
		TickerSymbol: symbol,
		Limit:        limit,
	})
	if err != nil {
		h.logger.Error("failed to get fin statements", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	statements := make([]*FinStatementResponse, 0, len(result))
	for _, s := range result {
		statements = append(statements, toFinStatementResponse(s))
	}

	respondJSON(w, h.logger, &GetFinStatementsResponse{Statements: statements})
}
