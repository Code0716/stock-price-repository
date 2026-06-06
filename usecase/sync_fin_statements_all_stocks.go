package usecase

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	syncFinStatementsAllStocksCheckpointRedisKey = "sync_fin_statements_all_stocks_checkpoint"
	// 週次運用で前回失敗のチェックポイントが翌週まで残らないよう短めTTL。
	syncFinStatementsAllStocksCheckpointRedisTTL = 2 * 24 * time.Hour
)

func (si *stockBrandInteractorImpl) SyncFinStatementsAllStocks(ctx context.Context, intervalMs, max int) error {
	brands, err := si.stockBrandRepository.FindAllMainMarkets(ctx)
	if err != nil {
		return errors.Wrap(err, "FindAllMainMarkets error")
	}

	checkpoint, err := si.readSyncFinStatementsAllStocksCheckpoint(ctx)
	if err != nil {
		return err
	}

	interval := time.Duration(intervalMs) * time.Millisecond
	processed, errCount := 0, 0
	for _, b := range brands {
		if max > 0 && processed >= max {
			break
		}
		if checkpoint != "" && b.TickerSymbol <= checkpoint {
			continue
		}

		if interval > 0 {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := si.SyncFinStatements(ctx, b.TickerSymbol); err != nil {
			log.Printf("SyncFinStatementsAllStocks: %s failed: %v", b.TickerSymbol, err)
			errCount++
		}

		if e := si.redisClient.Set(
			ctx,
			syncFinStatementsAllStocksCheckpointRedisKey,
			b.TickerSymbol,
			syncFinStatementsAllStocksCheckpointRedisTTL,
		).Err(); e != nil {
			log.Printf("SyncFinStatementsAllStocks: checkpoint save error: %v", e)
		}
		processed++
		if processed%50 == 0 {
			log.Printf("sync_fin_statements_all_stocks: processed %d/%d (errors=%d)", processed, len(brands), errCount)
		}
	}

	si.redisClient.Del(ctx, syncFinStatementsAllStocksCheckpointRedisKey)
	log.Printf("sync_fin_statements_all_stocks: completed. processed=%d errors=%d", processed, errCount)
	return nil
}

func (si *stockBrandInteractorImpl) readSyncFinStatementsAllStocksCheckpoint(ctx context.Context) (string, error) {
	checkpoint, err := si.redisClient.Get(ctx, syncFinStatementsAllStocksCheckpointRedisKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", errors.Wrap(err, "redisClient.Get error")
	}
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return checkpoint, nil
}
