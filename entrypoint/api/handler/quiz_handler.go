package handler

import (
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

type QuizHandler struct {
	usecase    usecase.QuizInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewQuizHandler(u usecase.QuizInteractor, s driver.HTTPServer, l *zap.Logger) *QuizHandler {
	return &QuizHandler{usecase: u, httpServer: s, logger: l}
}

// parseQuizDateParam クイズ対象日のクエリパラメータ(YYYY-MM-DD)をパースする。
func (h *QuizHandler) parseQuizDateParam(r *http.Request, key string) (*time.Time, error) {
	return h.httpServer.GetQueryParamDate(r, key, util.DateLayout)
}

func (h *QuizHandler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	date, err := h.parseQuizDateParam(r, "date")
	if err != nil {
		http.Error(w, "dateの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	questions, err := h.usecase.GetQuestions(r.Context(), date)
	if err != nil {
		h.logger.Error("quiz get questions failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, questions)
}

func (h *QuizHandler) GetChart(w http.ResponseWriter, r *http.Request) {
	quizDate, err := h.parseQuizDateParam(r, "quiz_date")
	if err != nil || quizDate == nil {
		http.Error(w, "quiz_dateは必須です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	stockBrandID := h.httpServer.GetQueryParam(r, "stock_brand_id")
	if stockBrandID == "" {
		http.Error(w, "stock_brand_idは必須です", http.StatusBadRequest)
		return
	}
	reveal := h.httpServer.GetQueryParam(r, "reveal") == "true"

	chart, err := h.usecase.GetChart(r.Context(), *quizDate, stockBrandID, reveal)
	if err != nil {
		if errors.Is(err, usecase.ErrQuizQuestionNotFound) {
			http.Error(w, "指定された設問が見つかりません", http.StatusNotFound)
			return
		}
		h.logger.Error("quiz get chart failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, chart)
}

type submitQuizAnswerRequest struct {
	QuizDate     string `json:"quizDate"`
	StockBrandID string `json:"stockBrandId"`
	Prediction   string `json:"prediction"`
}

func (h *QuizHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	var req submitQuizAnswerRequest
	if err := h.httpServer.ParseJSONBody(r, &req); err != nil {
		http.Error(w, "リクエストボディが不正です", http.StatusBadRequest)
		return
	}
	if req.QuizDate == "" || req.StockBrandID == "" || req.Prediction == "" {
		http.Error(w, "quizDate, stockBrandId, prediction は必須です", http.StatusBadRequest)
		return
	}

	prediction := models.QuizPrediction(req.Prediction)
	if !prediction.Valid() {
		http.Error(w, "predictionはstrong_down/down/up/strong_upのいずれかである必要があります", http.StatusBadRequest)
		return
	}

	quizDate, err := time.ParseInLocation(util.DateLayout, req.QuizDate, time.Local)
	if err != nil {
		http.Error(w, "quizDateの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	reveal, err := h.usecase.SubmitAnswer(r.Context(), quizDate, req.StockBrandID, prediction)
	if err != nil {
		if errors.Is(err, usecase.ErrQuizQuestionNotFound) {
			http.Error(w, "指定された設問が見つかりません", http.StatusNotFound)
			return
		}
		if errors.Is(err, repositories.ErrQuizAnswerAlreadyExists) {
			http.Error(w, "既に回答済みです", http.StatusConflict)
			return
		}
		h.logger.Error("quiz submit answer failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSONStatus(w, h.logger, http.StatusCreated, reveal)
}

func (h *QuizHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	quizDate, err := h.parseQuizDateParam(r, "date")
	if err != nil || quizDate == nil {
		http.Error(w, "dateは必須です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	results, err := h.usecase.GetResults(r.Context(), *quizDate)
	if err != nil {
		h.logger.Error("quiz get results failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, results)
}

func (h *QuizHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.usecase.GetStats(r.Context())
	if err != nil {
		h.logger.Error("quiz get stats failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, stats)
}
