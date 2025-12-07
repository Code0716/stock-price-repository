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
