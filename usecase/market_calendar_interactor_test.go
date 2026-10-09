package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestMarketCalendarInteractor_GetMarketCalendar(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 10, 31, 0, 0, 0, 0, time.Local)
	days := []*models.MarketCalendarDay{{Date: from, HolDiv: 1}}
	events := []*models.MarketEvent{{Date: from.AddDate(0, 0, 8), Kind: "sq_mini", Label: "SQ"}}

	type fields struct {
		repo func(ctrl *gomock.Controller) repositories.MarketCalendarRepository
	}
	tests := []struct {
		name    string
		fields  fields
		want    *MarketCalendar
		wantErr bool
	}{
		{
			name: "正常系: 営業日とイベントを返す",
			fields: fields{repo: func(ctrl *gomock.Controller) repositories.MarketCalendarRepository {
				m := mock_repositories.NewMockMarketCalendarRepository(ctrl)
				m.EXPECT().ListDays(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(days, nil)
				m.EXPECT().ListEvents(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(events, nil)
				return m
			}},
			want: &MarketCalendar{Days: days, Events: events},
		},
		{
			name: "異常系: ListDays 失敗",
			fields: fields{repo: func(ctrl *gomock.Controller) repositories.MarketCalendarRepository {
				m := mock_repositories.NewMockMarketCalendarRepository(ctrl)
				m.EXPECT().ListDays(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, errors.New("db"))
				return m
			}},
			wantErr: true,
		},
		{
			name: "異常系: ListEvents 失敗",
			fields: fields{repo: func(ctrl *gomock.Controller) repositories.MarketCalendarRepository {
				m := mock_repositories.NewMockMarketCalendarRepository(ctrl)
				m.EXPECT().ListDays(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(days, nil)
				m.EXPECT().ListEvents(gomock.Any(), gomock.Eq(from), gomock.Eq(to)).Return(nil, errors.New("db"))
				return m
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mi := NewMarketCalendarInteractor(tt.fields.repo(ctrl))
			got, err := mi.GetMarketCalendar(context.Background(), from, to)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetMarketCalendar() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
