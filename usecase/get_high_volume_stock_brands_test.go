package usecase

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	mock_repositories "github.com/Code0716/stock-price-repository/mock/repositories"
	"github.com/Code0716/stock-price-repository/models"
)

func TestGetHighVolumeStockBrandsInteractor_Execute(t *testing.T) {
	now := time.Now()

	type fields struct {
		highVolumeStockBrandRepo func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*models.HighVolumeStockBrand
		wantErr bool
	}{
		{
			name: "正常系: 全件取得成功",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindAll(gomock.Any()).
						Return([]*models.HighVolumeStockBrand{
							models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
							models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
						}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: []*models.HighVolumeStockBrand{
				models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
				models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
			},
			wantErr: false,
		},
		{
			name: "正常系: データが存在しない場合は空配列を返す",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindAll(gomock.Any()).
						Return([]*models.HighVolumeStockBrand{}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    []*models.HighVolumeStockBrand{},
			wantErr: false,
		},
		{
			name: "異常系: リポジトリエラー",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindAll(gomock.Any()).
						Return(nil, errors.New("database error"))
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var highVolumeStockBrandRepo *mock_repositories.MockHighVolumeStockBrandRepository
			if tt.fields.highVolumeStockBrandRepo != nil {
				highVolumeStockBrandRepo = tt.fields.highVolumeStockBrandRepo(ctrl)
			}

			u := &getHighVolumeStockBrandsInteractor{
				highVolumeStockBrandRepo: highVolumeStockBrandRepo,
			}

			got, err := u.Execute(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Execute() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHighVolumeStockBrandsInteractor_ExecuteWithPagination(t *testing.T) {
	now := time.Now()

	type fields struct {
		highVolumeStockBrandRepo func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository
	}
	type args struct {
		ctx        context.Context
		symbolFrom string
		limit      int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.PaginatedHighVolumeStockBrands
		wantErr bool
	}{
		{
			name: "正常系: limit=2で2件のみ存在する場合、NextCursorはnil",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(3)). // limit+1=3を要求
						Return([]*models.HighVolumeStockBrand{
							models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
							models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
						}, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbolFrom: "",
				limit:      2,
			},
			want: &models.PaginatedHighVolumeStockBrands{
				Brands: []*models.HighVolumeStockBrand{
					models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
					models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
				},
				NextCursor: nil,
				Limit:      2,
			},
			wantErr: false,
		},
		{
			name: "正常系: limit=2で3件以上存在する場合、2件だけ返されNextCursorが設定される",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(3)). // limit+1=3を要求
						Return([]*models.HighVolumeStockBrand{
							models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
							models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
							models.NewHighVolumeStockBrand("uuid-3", "1003", "Test Brand 3", 3000000, now),
						}, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbolFrom: "",
				limit:      2,
			},
			want: func() *models.PaginatedHighVolumeStockBrands {
				nextCursor := "1002"
				return &models.PaginatedHighVolumeStockBrands{
					Brands: []*models.HighVolumeStockBrand{
						models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
						models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
					},
					NextCursor: &nextCursor,
					Limit:      2,
				}
			}(),
			wantErr: false,
		},
		{
			name: "正常系: symbolFromを指定して次ページを取得",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindWithPagination(gomock.Any(), gomock.Eq("1002"), gomock.Eq(3)).
						Return([]*models.HighVolumeStockBrand{
							models.NewHighVolumeStockBrand("uuid-3", "1003", "Test Brand 3", 3000000, now),
							models.NewHighVolumeStockBrand("uuid-4", "1004", "Test Brand 4", 4000000, now),
							models.NewHighVolumeStockBrand("uuid-5", "1005", "Test Brand 5", 5000000, now),
						}, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbolFrom: "1002",
				limit:      2,
			},
			want: func() *models.PaginatedHighVolumeStockBrands {
				nextCursor := "1004"
				return &models.PaginatedHighVolumeStockBrands{
					Brands: []*models.HighVolumeStockBrand{
						models.NewHighVolumeStockBrand("uuid-3", "1003", "Test Brand 3", 3000000, now),
						models.NewHighVolumeStockBrand("uuid-4", "1004", "Test Brand 4", 4000000, now),
					},
					NextCursor: &nextCursor,
					Limit:      2,
				}
			}(),
			wantErr: false,
		},
		{
			name: "正常系: limit=0で全件取得（NextCursorは設定されない）",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(0)).
						Return([]*models.HighVolumeStockBrand{
							models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
							models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
							models.NewHighVolumeStockBrand("uuid-3", "1003", "Test Brand 3", 3000000, now),
						}, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbolFrom: "",
				limit:      0,
			},
			want: &models.PaginatedHighVolumeStockBrands{
				Brands: []*models.HighVolumeStockBrand{
					models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
					models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
					models.NewHighVolumeStockBrand("uuid-3", "1003", "Test Brand 3", 3000000, now),
				},
				NextCursor: nil,
				Limit:      0,
			},
			wantErr: false,
		},
		{
			name: "正常系: データが存在しない場合は空配列とNextCursor=nilを返す",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(3)).
						Return([]*models.HighVolumeStockBrand{}, nil)
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbolFrom: "",
				limit:      2,
			},
			want: &models.PaginatedHighVolumeStockBrands{
				Brands:     []*models.HighVolumeStockBrand{},
				NextCursor: nil,
				Limit:      2,
			},
			wantErr: false,
		},
		{
			name: "異常系: リポジトリエラー",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					mock := mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
					mock.EXPECT().
						FindWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(3)).
						Return(nil, errors.New("database error"))
					return mock
				},
			},
			args: args{
				ctx:        context.Background(),
				symbolFrom: "",
				limit:      2,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "異常系: limitが負の値の場合はエラー",
			fields: fields{
				highVolumeStockBrandRepo: func(ctrl *gomock.Controller) *mock_repositories.MockHighVolumeStockBrandRepository {
					// 負の値の場合、リポジトリは呼び出されない
					return mock_repositories.NewMockHighVolumeStockBrandRepository(ctrl)
				},
			},
			args: args{
				ctx:        context.Background(),
				symbolFrom: "",
				limit:      -1,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var highVolumeStockBrandRepo *mock_repositories.MockHighVolumeStockBrandRepository
			if tt.fields.highVolumeStockBrandRepo != nil {
				highVolumeStockBrandRepo = tt.fields.highVolumeStockBrandRepo(ctrl)
			}

			u := &getHighVolumeStockBrandsInteractor{
				highVolumeStockBrandRepo: highVolumeStockBrandRepo,
			}

			got, err := u.ExecuteWithPagination(tt.args.ctx, tt.args.symbolFrom, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteWithPagination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExecuteWithPagination() got = %v, want %v", got, tt.want)
			}
		})
	}
}
