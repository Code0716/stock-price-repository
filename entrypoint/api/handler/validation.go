package handler

import (
	"net/http"
	"regexp"
	"time"

	"github.com/Code0716/stock-price-repository/util"
)

var (
	// alphanumericRequiredRegex 英数字1文字以上のパターン（必須フィールド用）
	alphanumericRequiredRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	// alphanumericOptionalRegex 英数字0文字以上のパターン（任意フィールド用）
	alphanumericOptionalRegex = regexp.MustCompile(`^[a-zA-Z0-9]*$`)
)

// parseDateRange from/to クエリを解析し from<=to を検証する。
// from/to はいずれも省略可能で、指定されなければ nil を返す。
func parseDateRange(r *http.Request) (from, to *time.Time, err error) {
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, parseErr := time.ParseInLocation(util.DateLayout, fromStr, time.Local)
		if parseErr != nil {
			return nil, nil, &validationError{message: "fromの日付形式が不正です (YYYY-MM-DD)"}
		}
		from = &t
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, parseErr := time.ParseInLocation(util.DateLayout, toStr, time.Local)
		if parseErr != nil {
			return nil, nil, &validationError{message: "toの日付形式が不正です (YYYY-MM-DD)"}
		}
		to = &t
	}

	if from != nil && to != nil && from.After(*to) {
		return nil, nil, &validationError{message: "fromはto以前の日付である必要があります"}
	}

	return from, to, nil
}
