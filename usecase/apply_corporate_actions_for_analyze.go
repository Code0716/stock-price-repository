package usecase

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/models"
)

// analyzeCorporateActionLookbackYears - 権利落ちを検知した銘柄について、for_analyzeを
// 再取得・上書きする際にどこまで遡るか。J-Quants Lightプランの上限（日足5年）に合わせる。
const analyzeCorporateActionLookbackYears = 5

// applyCorporateActionsForAnalyze - 日次取込で取得した当日分の日足からJ-Quantsの
// AdjustmentFactor(分割・併合の調整係数)を検査し、権利落ちが発生した銘柄については
// for_analyzeの当該銘柄の全期間をJ-Quantsの調整済みOHLCV(Adjustment*)で再取得・上書きする。
//
// J-Quantsは指定期間のAdjustment*系列を「クエリ時点で」累積調整済みの値として返すため、
// こちらで価格×比率のような遡及計算をする必要はない（権利落ち日・比率の判定含めてJ-Quants任せ）。
// これにより、従来の「手書きの分割リストをCLIで流す」運用で発生していた
// split_dateの手入力ミス・adj_close_priceの二重調整が構造的に起きなくなる。
//
// 冪等性: (symbol, effectiveDate) を applied_stock_splits_history /
// applied_stock_consolidations_history に記録し、二重適用を避ける。この記録は
// クイズ採点(grade_quiz_answers)・買い候補評価(evaluate_daily_stock_picks)の
// void判定にも使われるため、正しい権利落ち日で記録されることが重要。
func (si *stockBrandsDailyStockPriceInteractorImpl) applyCorporateActionsForAnalyze(
	ctx context.Context,
	gatewayPrices []*gateway.StockPrice,
	now time.Time,
) error {
	one := decimal.NewFromInt(1)

	for _, v := range gatewayPrices {
		if v == nil {
			continue
		}
		// AdjustmentFactorが未設定(ゼロ値)、または1（無調整）の銘柄はスキップする。
		if v.AdjustmentFactor.IsZero() || v.AdjustmentFactor.Equal(one) {
			continue
		}

		if err := si.applyCorporateActionForSymbol(ctx, v.TickerSymbol, v.Date, v.AdjustmentFactor, now); err != nil {
			// 1銘柄の失敗で日次バッチ全体を止めない。ログに残して次の銘柄へ進む。
			log.Printf("applyCorporateActionForSymbol error: symbol=%s date=%s: %v", v.TickerSymbol, v.Date.Format("2006-01-02"), err)
		}
	}

	return nil
}

// applyCorporateActionForSymbol - 1銘柄について権利落ちを検知した際の再取得・上書き・履歴記録を行う。
// 外部APIコールはトランザクションの外で行い、DB書き込みのみ tx.DoInTx でラップする。
func (si *stockBrandsDailyStockPriceInteractorImpl) applyCorporateActionForSymbol(
	ctx context.Context,
	symbol string,
	effectiveDate time.Time,
	adjustmentFactor decimal.Decimal,
	now time.Time,
) error {
	isSplit := adjustmentFactor.LessThan(decimal.NewFromInt(1))

	alreadyApplied, err := si.corporateActionAlreadyApplied(ctx, symbol, effectiveDate, isSplit)
	if err != nil {
		return errors.Wrap(err, "corporateActionAlreadyApplied error")
	}
	if alreadyApplied {
		return nil
	}

	// J-Quantsから当該銘柄の調整済みOHLCVを取得しなおす（トランザクション外）。
	dateFrom := now.AddDate(-analyzeCorporateActionLookbackYears, 0, 0)
	prices, err := si.stockAPIClient.GetDailyPricesBySymbolAndRange(ctx, gateway.StockAPISymbol(symbol), dateFrom, now)
	if err != nil {
		return errors.Wrap(err, "stockAPIClient.GetDailyPricesBySymbolAndRange error")
	}
	if len(prices) == 0 {
		log.Printf("no historical prices returned for symbol=%s", symbol)
		return nil
	}

	analyzePrices := newStockBrandDailyPriceForAnalyzeFromGatewayPrices(prices, now)

	err = si.tx.DoInTx(ctx, func(ctx context.Context) error {
		if err := si.stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze(ctx, analyzePrices); err != nil {
			return errors.Wrap(err, "stockBrandsDailyPriceForAnalyzeRepository.CreateStockBrandDailyPriceForAnalyze error")
		}

		if err := si.recordCorporateAction(ctx, symbol, effectiveDate, adjustmentFactor, isSplit); err != nil {
			return errors.Wrap(err, "recordCorporateAction error")
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "DoInTx error")
	}

	log.Printf("applied corporate action: symbol=%s effectiveDate=%s adjustmentFactor=%s rows=%d",
		symbol, effectiveDate.Format("2006-01-02"), adjustmentFactor.String(), len(analyzePrices))

	return nil
}

func (si *stockBrandsDailyStockPriceInteractorImpl) corporateActionAlreadyApplied(
	ctx context.Context,
	symbol string,
	effectiveDate time.Time,
	isSplit bool,
) (bool, error) {
	if isSplit {
		return si.appliedStockSplitsHistoryRepository.Exists(ctx, symbol, effectiveDate)
	}
	return si.appliedStockConsolidationsHistoryRepository.Exists(ctx, symbol, effectiveDate)
}

func (si *stockBrandsDailyStockPriceInteractorImpl) recordCorporateAction(
	ctx context.Context,
	symbol string,
	effectiveDate time.Time,
	adjustmentFactor decimal.Decimal,
	isSplit bool,
) error {
	if isSplit {
		// 既存テーブルの ratio は「分割前株数/分割後株数」相当（例: 1:2分割で2.0）。
		// AdjustmentFactorは逆数（1:2分割で0.5）にあたるため変換する。
		ratio := decimal.NewFromInt(1).Div(adjustmentFactor)
		history := models.NewAppliedStockSplitHistory(symbol, effectiveDate, ratio)
		return si.appliedStockSplitsHistoryRepository.Create(ctx, history)
	}

	// 併合(逆分割)は AdjustmentFactor がそのまま「旧株数/新株数」に相当する。
	history := models.NewAppliedStockConsolidationHistory(symbol, effectiveDate, adjustmentFactor)
	return si.appliedStockConsolidationsHistoryRepository.Create(ctx, history)
}
