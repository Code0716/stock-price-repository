package handler

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

// marketCalendarMaxRangeDays 1リクエストで取得できる最大日数（カレンダー表示は1〜数か月分なので十分）。
const marketCalendarMaxRangeDays = 400

type MarketCalendarHandler struct {
	usecase usecase.MarketCalendarInteractor
	logger  *zap.Logger
}

func NewMarketCalendarHandler(u usecase.MarketCalendarInteractor, l *zap.Logger) *MarketCalendarHandler {
	return &MarketCalendarHandler{usecase: u, logger: l}
}

type marketCalendarDayResponse struct {
	Date   string `json:"date"`
	HolDiv int    `json:"holDiv"`
}

type marketCalendarEventResponse struct {
	Date  string `json:"date"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type marketCalendarResponse struct {
	Days   []marketCalendarDayResponse   `json:"days"`
	Events []marketCalendarEventResponse `json:"events"`
}

// GetMarketCalendar GET /market/calendar?from=YYYY-MM-DD&to=YYYY-MM-DD
// 営業日・休場日(holDiv: 0休場/1営業/2半日/3休場だが祝日取引あり)と市場イベント(SQ・日銀・FOMC・米CPI・米雇用統計)を返す。
// from, to は必須。該当データが無ければ 200 で空配列を返す。
func (h *MarketCalendarHandler) GetMarketCalendar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		writeError(w, h.logger, "market calendar parse date range failed", err)
		return
	}
	if from == nil || to == nil {
		http.Error(w, "from と to は必須です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if to.Sub(*from).Hours()/24 > marketCalendarMaxRangeDays {
		http.Error(w, "from と to の間隔は400日以内で指定してください", http.StatusBadRequest)
		return
	}

	cal, err := h.usecase.GetMarketCalendar(r.Context(), *from, *to)
	if err != nil {
		writeError(w, h.logger, "market calendar get failed", err)
		return
	}

	res := marketCalendarResponse{
		Days:   make([]marketCalendarDayResponse, 0, len(cal.Days)),
		Events: make([]marketCalendarEventResponse, 0, len(cal.Events)),
	}
	for _, d := range cal.Days {
		res.Days = append(res.Days, marketCalendarDayResponse{Date: d.Date.Format(util.DateLayout), HolDiv: d.HolDiv})
	}
	for _, e := range cal.Events {
		res.Events = append(res.Events, marketCalendarEventResponse{Date: e.Date.Format(util.DateLayout), Kind: e.Kind, Label: e.Label})
	}
	respondJSON(w, h.logger, res)
}
