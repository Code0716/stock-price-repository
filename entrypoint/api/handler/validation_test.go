package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Code0716/stock-price-repository/util"
	"github.com/stretchr/testify/assert"
)

func TestParseDateRange(t *testing.T) {
	from, _ := time.ParseInLocation(util.DateLayout, "2024-01-01", time.Local)
	to, _ := time.ParseInLocation(util.DateLayout, "2024-01-31", time.Local)

	tests := []struct {
		name       string
		url        string
		wantFrom   *time.Time
		wantTo     *time.Time
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "正常系: from/to とも指定",
			url:      "/?from=2024-01-01&to=2024-01-31",
			wantFrom: &from,
			wantTo:   &to,
		},
		{
			name:     "正常系: from のみ指定",
			url:      "/?from=2024-01-01",
			wantFrom: &from,
			wantTo:   nil,
		},
		{
			name:     "正常系: to のみ指定",
			url:      "/?to=2024-01-31",
			wantFrom: nil,
			wantTo:   &to,
		},
		{
			name:     "正常系: from/to とも省略",
			url:      "/",
			wantFrom: nil,
			wantTo:   nil,
		},
		{
			name:       "異常系: from の日付形式が不正",
			url:        "/?from=invalid",
			wantErr:    true,
			wantErrMsg: "fromの日付形式が不正です (YYYY-MM-DD)",
		},
		{
			name:       "異常系: to の日付形式が不正",
			url:        "/?to=invalid",
			wantErr:    true,
			wantErrMsg: "toの日付形式が不正です (YYYY-MM-DD)",
		},
		{
			name:       "異常系: from が to より後",
			url:        "/?from=2024-01-31&to=2024-01-01",
			wantErr:    true,
			wantErrMsg: "fromはto以前の日付である必要があります",
		},
		{
			name:     "正常系: from と to が同一日",
			url:      "/?from=2024-01-01&to=2024-01-01",
			wantFrom: &from,
			wantTo:   &from,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.url, nil)

			gotFrom, gotTo, err := parseDateRange(r)

			if tt.wantErr {
				assert.Error(t, err)
				var verr *validationError
				assert.ErrorAs(t, err, &verr)
				assert.Equal(t, tt.wantErrMsg, err.Error())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantFrom, gotFrom)
			assert.Equal(t, tt.wantTo, gotTo)
		})
	}
}
