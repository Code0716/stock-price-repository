package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormtests "gorm.io/gorm/utils/tests"

	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
)

func newTestGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormtests.DummyDialector{}, &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestTxOrDefault(t *testing.T) {
	fallbackDB := newTestGormDB(t)
	fallbackQuery := genQuery.Use(fallbackDB)

	txDB := newTestGormDB(t)

	tests := []struct {
		name     string
		ctx      func() context.Context
		wantDB   *gorm.DB
		wantSame bool
	}{
		{
			name: "ctxにtxが無い場合はqをそのまま返す",
			ctx: func() context.Context {
				return context.Background()
			},
			wantDB:   fallbackDB,
			wantSame: true,
		},
		{
			name: "ctxにtxがある場合はtx由来のqueryを返す",
			ctx: func() context.Context {
				return setTx(context.Background(), txDB)
			},
			wantDB:   txDB,
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TxOrDefault(tt.ctx(), fallbackQuery)

			assert.Equal(t, tt.wantDB, got.UnderlyingDB())
			if tt.wantSame {
				assert.Same(t, fallbackQuery, got)
			} else {
				assert.NotSame(t, fallbackQuery, got)
			}
		})
	}
}
