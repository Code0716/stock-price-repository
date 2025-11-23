package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

func TestStockBrandInteractorImpl_UpdateStockBrands(t *testing.T) {
	type fields struct {
		tx                                        func(ctrl *gomock.Controller) repositories.Transaction
		stockBrandRepository                      func(ctrl *gomock.Controller) repositories.StockBrandRepository
		stockBrandsDailyPriceRepository           func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
		analyzeStockBrandPriceHistoryRepository   func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository
		stockBrandsDailyPriceForAnalyzeRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
		stockAPIClient                            func(ctrl *gomock.Controller) gateway.StockAPIClient
	}
	type args struct {
		ctx context.Context
		now time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success - No delisting",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetStockBrands(gomock.Any()).Return([]*gateway.StockBrand{
						{
							Symbol:           "1111",
							CompanyName:      "Test Company",
							MarketCode:       "P",
							MarketCodeName:   "Prime",
							Sector33Code:     "1000",
							Sector33CodeName: "Sector",
							Sector17Code:     "10",
							Sector17CodeName: "Sector17",
						},
					}, nil)
					return mock
				},
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindAll(gomock.Any()).Return([]*models.StockBrand{}, nil)
					mock.EXPECT().UpsertStockBrands(gomock.Any(), gomock.Any()).Return(nil)
					mock.EXPECT().FindDelistingStockBrandsFromUpdateTime(gomock.Any(), gomock.Any()).Return([]string{}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "Success - With delisting",
			fields: fields{
				tx: func(ctrl *gomock.Controller) repositories.Transaction {
					mock := mock_repositories.NewMockTransaction(ctrl)
					mock.EXPECT().DoInTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})
					return mock
				},
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetStockBrands(gomock.Any()).Return([]*gateway.StockBrand{
						{
							Symbol:           "1111",
							CompanyName:      "Test Company",
							MarketCode:       "P",
							MarketCodeName:   "Prime",
							Sector33Code:     "1000",
							Sector33CodeName: "Sector",
							Sector17Code:     "10",
							Sector17CodeName: "Sector17",
						},
					}, nil)
					return mock
				},
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().FindAll(gomock.Any()).Return([]*models.StockBrand{}, nil)
					mock.EXPECT().UpsertStockBrands(gomock.Any(), gomock.Any()).Return(nil)
					mock.EXPECT().FindDelistingStockBrandsFromUpdateTime(gomock.Any(), gomock.Any()).Return([]string{"999"}, nil)
					mock.EXPECT().DeleteDelistingStockBrands(gomock.Any(), []string{"999"}).Return([]*models.StockBrand{
						{
							TickerSymbol: "9999",
						},
					}, nil)
					return mock
				},
				stockBrandsDailyPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().DeleteByIDs(gomock.Any(), []string{"999"}).Return(nil)
					return mock
				},
				analyzeStockBrandPriceHistoryRepository: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					mock := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					mock.EXPECT().DeleteByStockBrandIDs(gomock.Any(), []string{"999"}).Return(nil)
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().DeleteBySymbols(gomock.Any(), []string{"9999"}).Return(nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := &stockBrandInteractorImpl{}
			if tt.fields.tx != nil {
				s.tx = tt.fields.tx(ctrl)
			}
			if tt.fields.stockBrandRepository != nil {
				s.stockBrandRepository = tt.fields.stockBrandRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyPriceRepository != nil {
				s.stockBrandsDailyPriceRepository = tt.fields.stockBrandsDailyPriceRepository(ctrl)
			}
			if tt.fields.analyzeStockBrandPriceHistoryRepository != nil {
				s.analyzeStockBrandPriceHistoryRepository = tt.fields.analyzeStockBrandPriceHistoryRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyPriceForAnalyzeRepository != nil {
				s.stockBrandsDailyPriceForAnalyzeRepository = tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl)
			}
			if tt.fields.stockAPIClient != nil {
				s.stockAPIClient = tt.fields.stockAPIClient(ctrl)
			}

			if err := s.UpdateStockBrands(tt.args.ctx, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("StockBrandInteractorImpl.UpdateStockBrands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockBrandInteractorImpl_matchStockBrandIDs(t *testing.T) {
	type args struct {
		newBrands     []*models.StockBrand
		currentBrands []*models.StockBrand
	}
	tests := []struct {
		name string
		args args
		want []*models.StockBrand
	}{
		{
			name: "Match found",
			args: args{
				newBrands: []*models.StockBrand{
					{TickerSymbol: "1111", ID: ""},
				},
				currentBrands: []*models.StockBrand{
					{TickerSymbol: "1111", ID: "existing-id"},
				},
			},
			want: []*models.StockBrand{
				{TickerSymbol: "1111", ID: "existing-id"},
			},
		},
		{
			name: "No match found",
			args: args{
				newBrands: []*models.StockBrand{
					{TickerSymbol: "2222", ID: ""},
				},
				currentBrands: []*models.StockBrand{
					{TickerSymbol: "1111", ID: "existing-id"},
				},
			},
			want: []*models.StockBrand{
				{TickerSymbol: "2222", ID: ""},
			},
		},
		{
			name: "Empty current brands",
			args: args{
				newBrands: []*models.StockBrand{
					{TickerSymbol: "1111", ID: ""},
				},
				currentBrands: []*models.StockBrand{},
			},
			want: []*models.StockBrand{
				{TickerSymbol: "1111", ID: ""},
			},
		},
		{
			name: "Empty new brands",
			args: args{
				newBrands:     []*models.StockBrand{},
				currentBrands: []*models.StockBrand{{TickerSymbol: "1111", ID: "id"}},
			},
			want: []*models.StockBrand{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			si := &stockBrandInteractorImpl{}
			si.matchStockBrandIDs(tt.args.newBrands, tt.args.currentBrands)

			for i, brand := range tt.args.newBrands {
				if brand.ID != tt.want[i].ID {
					t.Errorf("matchStockBrandIDs() brand.ID = %v, want %v", brand.ID, tt.want[i].ID)
				}
			}
		})
	}
}

func TestStockBrandInteractorImpl_handleDelistedStockBrands(t *testing.T) {
	type fields struct {
		stockBrandRepository                      func(ctrl *gomock.Controller) repositories.StockBrandRepository
		stockBrandsDailyPriceRepository           func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository
		analyzeStockBrandPriceHistoryRepository   func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository
		stockBrandsDailyPriceForAnalyzeRepository func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository
	}
	type args struct {
		ctx       context.Context
		deleteIDs []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "Success - No delete IDs",
			fields: fields{},
			args: args{
				ctx:       context.Background(),
				deleteIDs: []string{},
			},
			wantErr: false,
		},
		{
			name: "Success - With delete IDs",
			fields: fields{
				stockBrandsDailyPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().DeleteByIDs(gomock.Any(), []string{"1"}).Return(nil)
					return mock
				},
				analyzeStockBrandPriceHistoryRepository: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					mock := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					mock.EXPECT().DeleteByStockBrandIDs(gomock.Any(), []string{"1"}).Return(nil)
					return mock
				},
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().DeleteDelistingStockBrands(gomock.Any(), []string{"1"}).Return([]*models.StockBrand{
						{TickerSymbol: "1111"},
					}, nil)
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().DeleteBySymbols(gomock.Any(), []string{"1111"}).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				deleteIDs: []string{"1"},
			},
			wantErr: false,
		},
		{
			name: "Error - stockBrandsDailyPriceRepository.DeleteByIDs",
			fields: fields{
				stockBrandsDailyPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().DeleteByIDs(gomock.Any(), []string{"1"}).Return(errors.New("error"))
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				deleteIDs: []string{"1"},
			},
			wantErr: true,
		},
		{
			name: "Error - analyzeStockBrandPriceHistoryRepository.DeleteByStockBrandIDs",
			fields: fields{
				stockBrandsDailyPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().DeleteByIDs(gomock.Any(), []string{"1"}).Return(nil)
					return mock
				},
				analyzeStockBrandPriceHistoryRepository: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					mock := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					mock.EXPECT().DeleteByStockBrandIDs(gomock.Any(), []string{"1"}).Return(errors.New("error"))
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				deleteIDs: []string{"1"},
			},
			wantErr: true,
		},
		{
			name: "Error - stockBrandRepository.DeleteDelistingStockBrands",
			fields: fields{
				stockBrandsDailyPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().DeleteByIDs(gomock.Any(), []string{"1"}).Return(nil)
					return mock
				},
				analyzeStockBrandPriceHistoryRepository: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					mock := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					mock.EXPECT().DeleteByStockBrandIDs(gomock.Any(), []string{"1"}).Return(nil)
					return mock
				},
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().DeleteDelistingStockBrands(gomock.Any(), []string{"1"}).Return(nil, errors.New("error"))
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				deleteIDs: []string{"1"},
			},
			wantErr: true,
		},
		{
			name: "Error - stockBrandsDailyPriceForAnalyzeRepository.DeleteBySymbols",
			fields: fields{
				stockBrandsDailyPriceRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceRepository(ctrl)
					mock.EXPECT().DeleteByIDs(gomock.Any(), []string{"1"}).Return(nil)
					return mock
				},
				analyzeStockBrandPriceHistoryRepository: func(ctrl *gomock.Controller) repositories.AnalyzeStockBrandPriceHistoryRepository {
					mock := mock_repositories.NewMockAnalyzeStockBrandPriceHistoryRepository(ctrl)
					mock.EXPECT().DeleteByStockBrandIDs(gomock.Any(), []string{"1"}).Return(nil)
					return mock
				},
				stockBrandRepository: func(ctrl *gomock.Controller) repositories.StockBrandRepository {
					mock := mock_repositories.NewMockStockBrandRepository(ctrl)
					mock.EXPECT().DeleteDelistingStockBrands(gomock.Any(), []string{"1"}).Return([]*models.StockBrand{
						{TickerSymbol: "1111"},
					}, nil)
					return mock
				},
				stockBrandsDailyPriceForAnalyzeRepository: func(ctrl *gomock.Controller) repositories.StockBrandsDailyPriceForAnalyzeRepository {
					mock := mock_repositories.NewMockStockBrandsDailyPriceForAnalyzeRepository(ctrl)
					mock.EXPECT().DeleteBySymbols(gomock.Any(), []string{"1111"}).Return(errors.New("error"))
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				deleteIDs: []string{"1"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			si := &stockBrandInteractorImpl{}
			if tt.fields.stockBrandRepository != nil {
				si.stockBrandRepository = tt.fields.stockBrandRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyPriceRepository != nil {
				si.stockBrandsDailyPriceRepository = tt.fields.stockBrandsDailyPriceRepository(ctrl)
			}
			if tt.fields.analyzeStockBrandPriceHistoryRepository != nil {
				si.analyzeStockBrandPriceHistoryRepository = tt.fields.analyzeStockBrandPriceHistoryRepository(ctrl)
			}
			if tt.fields.stockBrandsDailyPriceForAnalyzeRepository != nil {
				si.stockBrandsDailyPriceForAnalyzeRepository = tt.fields.stockBrandsDailyPriceForAnalyzeRepository(ctrl)
			}

			if err := si.handleDelistedStockBrands(tt.args.ctx, tt.args.deleteIDs); (err != nil) != tt.wantErr {
				t.Errorf("StockBrandInteractorImpl.handleDelistedStockBrands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
