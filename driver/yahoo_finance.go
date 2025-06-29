//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/pkg/errors"
)

// 週足以外
func (c *StockAPIClient) getStockPriceChart(
	ctx context.Context,
	symbol gateway.StockAPISymbol,
	interval gateway.StockAPIInterval,
	dateRange gateway.StockAPIValidRange,
) (*gateway.StockChartWithRangeAPIResponseInfo, error) {
	u, err := url.Parse(
		fmt.Sprintf("%s/v8/finance/chart/%s?interval=%s&range=%s",
			config.YahooFinance().BaseURL,
			symbol,
			interval.String(),
			dateRange.String(),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, u.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "yahoo.finance.api request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", "SttAppUserAgent/1.0") // 形式変えるとエラーになるのでとりあえずこれに。
	res, err := c.request.GetHttpClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`yahoo.finance.api request to: %s`, u.String()))
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "yahoo.finance.api io.ReadAll error")
	}
	if res.StatusCode != http.StatusOK {
		// エラー時のResponseが不明なので、log出してエラーを返す。
		// ハンドリングしたい気持ちはある。
		return nil, errors.New(fmt.Sprintf(`yahoo.finance.api status error status: %d, url: %s`, res.StatusCode, u.String()))
	}

	var response yahooFinanceAPIResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`yahoo.finance.api request to: %s`, u.String()))
	}

	responseInfo := c.yahooFinanceAPIResponseToResponseInfo(response, interval, dateRange)
	return responseInfo, nil
}

func (c *StockAPIClient) yahooFinanceAPIResponseToResponseInfo(
	response yahooFinanceAPIResponse,
	interval gateway.StockAPIInterval,
	dateRange gateway.StockAPIValidRange,
) *gateway.StockChartWithRangeAPIResponseInfo {
	symbol := c.symbolSanitizeToStr(response.Chart.Result[0].Meta.Symbol)
	return &gateway.StockChartWithRangeAPIResponseInfo{
		TickerSymbol:    symbol,
		InstrumentType:  response.Chart.Result[0].Meta.InstrumentType,
		DataGranularity: response.Chart.Result[0].Meta.DataGranularity,
		Range:           response.Chart.Result[0].Meta.Range,
		Indicator: c.yahooFinanceAPIResponseIndicatorToResponseInfoIndicator(
			symbol,
			response.Chart.Result[0].Timestamp,
			response.Chart.Result[0].Indicators,
			interval,
			dateRange,
		),
	}
}

func (c *StockAPIClient) yahooFinanceAPIResponseIndicatorToResponseInfoIndicator(
	symbol string,
	timestamp []int64,
	indicators YahooFinanceIndicators,
	interval gateway.StockAPIInterval,
	dateRange gateway.StockAPIValidRange,
) []*gateway.StockPrice {
	items := make([]*gateway.StockPrice, 0, len(timestamp))
	weekCount := int64(len(timestamp))
	// nヶ月の週足取得すると、今週の月曜日が祝日だった際に、別の週とカウントされ、1週多く返される。
	// そのため今週は返さない。
	if interval == gateway.StockAPIInterval1WK {
		// 日付だけの比較だし、外から与えて変な週数になるのを避けたいのでここでtime.Now()を呼び出す。
		now := time.Now()
		switch dateRange {
		case gateway.StockAPIValidRange1MO:
			weekCount = c.calcWeekCount(now, 1)
		case gateway.StockAPIValidRange3MO:
			weekCount = c.calcWeekCount(now, 3)
		case gateway.StockAPIValidRange6MO:
			weekCount = c.calcWeekCount(now, 6)
		}
		if weekCount != 0 {
			items = make([]*gateway.StockPrice, 0, weekCount)
		}
	}

	for i, v := range timestamp[:weekCount] {
		adjustmentClose := indicators.Quote[0].Close[i]
		if indicators.Adjclose != nil {
			adjustmentClose = indicators.Adjclose[0].Adjclose[i]
		}
		items = append(items, &gateway.StockPrice{
			TickerSymbol:    symbol,
			Date:            util.UnixToDatetime(v),
			High:            indicators.Quote[0].High[i],
			Low:             indicators.Quote[0].Low[i],
			Open:            indicators.Quote[0].Open[i],
			Close:           indicators.Quote[0].Close[i],
			Volume:          indicators.Quote[0].Volume[i],
			AdjustmentClose: adjustmentClose,
		})
	}
	return items
}

// 週足
func (c *StockAPIClient) GetWeeklyIndexPriceChart(
	ctx context.Context,
	symbol gateway.StockAPISymbol,
	dateRange gateway.StockAPIValidRange,
) (*gateway.StockChartWithRangeAPIResponseInfo, error) {
	weeklyResponse, err := c.getStockPriceChart(ctx, symbol, gateway.StockAPIInterval1WK, dateRange)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	// 現在値(close)の取得
	// 3ヶ月の週足取得すると、今週の月曜日が祝日だった際に、別の週とカウントされ、1週多く返される。
	// そのため週足では今週を返してないので、現在値を取得して追加してあげる。
	todayChart, err := c.getStockPriceChart(ctx, symbol, gateway.StockAPIInterval1D, gateway.StockAPIValidRange1D)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	weeklyResponse.Indicator = append(weeklyResponse.Indicator, todayChart.Indicator[0])
	return weeklyResponse, nil
}

// 週の開始日を月曜日としてnヶ月が何週か取得する
// 今週は含まない
func (c *StockAPIClient) calcWeekCount(today time.Time, monthsAgo int) int64 {
	// nヶ月前の日付を計算
	threeMonthsAgo := today.AddDate(0, -monthsAgo, 0)

	// 現在の週の開始日を取得
	startOfWeek := today.AddDate(0, 0, -int(today.Weekday()-time.Monday))

	// nヶ月前の週の開始日を取得
	endOfWeek := threeMonthsAgo.AddDate(0, 0, -int(threeMonthsAgo.Weekday()-time.Monday))

	// 週数を計算して返す
	return int64(startOfWeek.Sub(endOfWeek).Hours() / (24 * 7))
}

func (c *StockAPIClient) yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo(bs YahooFinanceBalanceSheetsResponse) (*gateway.BalanceSheetsInfo, error) {
	var bsItem []*gateway.BalanceSheetItem
	for _, v := range bs.BalanceSheets {
		date, err := c.parseBalanceSheetsDateToJST(v.Date)
		if err != nil {
			log.Printf("error parsing time date in yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo: %v", err)
			return nil, errors.Wrap(err, "error parsing time date in yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo")
		}

		bsItem = append(bsItem, &gateway.BalanceSheetItem{
			Date: *date,
			BalanceSheet: gateway.BalanceSheet{
				TreasurySharesNumber:                v.BalanceSheet.TreasurySharesNumber,
				ShareIssued:                         v.BalanceSheet.ShareIssued,
				TotalDebt:                           v.BalanceSheet.TotalDebt,
				TangibleBookValue:                   v.BalanceSheet.TangibleBookValue,
				InvestedCapital:                     v.BalanceSheet.InvestedCapital,
				WorkingCapital:                      v.BalanceSheet.WorkingCapital,
				NetTangibleAssets:                   v.BalanceSheet.NetTangibleAssets,
				CapitalLeaseObligations:             v.BalanceSheet.CapitalLeaseObligations,
				CommonStockEquity:                   v.BalanceSheet.CommonStockEquity,
				TotalCapitalization:                 v.BalanceSheet.TotalCapitalization,
				TotalEquityGrossMinorityInterest:    v.BalanceSheet.TotalEquityGrossMinorityInterest,
				MinorityInterest:                    v.BalanceSheet.MinorityInterest,
				StockholdersEquity:                  v.BalanceSheet.StockholdersEquity,
				TreasuryStock:                       v.BalanceSheet.TreasuryStock,
				RetainedEarnings:                    v.BalanceSheet.RetainedEarnings,
				AdditionalPaidInCapital:             v.BalanceSheet.AdditionalPaidInCapital,
				CapitalStock:                        v.BalanceSheet.CapitalStock,
				CommonStock:                         v.BalanceSheet.CommonStock,
				TotalLiabilitiesNetMinorityInterest: v.BalanceSheet.TotalLiabilitiesNetMinorityInterest,
				TotalNonCurrentLiabilitiesNetMinorityInterest:       v.BalanceSheet.TotalNonCurrentLiabilitiesNetMinorityInterest,
				OtherNonCurrentLiabilities:                          v.BalanceSheet.OtherNonCurrentLiabilities,
				NonCurrentPensionAndOtherPostretirementBenefitPlans: v.BalanceSheet.NonCurrentPensionAndOtherPostretirementBenefitPlans,
				NonCurrentDeferredTaxesLiabilities:                  v.BalanceSheet.NonCurrentDeferredTaxesLiabilities,
				LongTermDebtAndCapitalLeaseObligation:               v.BalanceSheet.LongTermDebtAndCapitalLeaseObligation,
				LongTermCapitalLeaseObligation:                      v.BalanceSheet.LongTermCapitalLeaseObligation,
				LongTermDebt:                                        v.BalanceSheet.LongTermDebt,
				LongTermProvisions:                                  v.BalanceSheet.LongTermProvisions,
				CurrentLiabilities:                                  v.BalanceSheet.CurrentLiabilities,
				OtherCurrentLiabilities:                             v.BalanceSheet.OtherCurrentLiabilities,
				CurrentDebtAndCapitalLeaseObligation:                v.BalanceSheet.CurrentDebtAndCapitalLeaseObligation,
				CurrentDebt:                                         v.BalanceSheet.CurrentDebt,
				PensionandOtherPostRetirementBenefitPlansCurrent:    v.BalanceSheet.PensionandOtherPostRetirementBenefitPlansCurrent,
				CurrentProvisions:                                   v.BalanceSheet.CurrentProvisions,
				Payables:                                            v.BalanceSheet.Payables,
				TotalTaxPayable:                                     v.BalanceSheet.TotalTaxPayable,
				AccountsPayable:                                     v.BalanceSheet.AccountsPayable,
				TotalAssets:                                         v.BalanceSheet.TotalAssets,
				TotalNonCurrentAssets:                               v.BalanceSheet.TotalNonCurrentAssets,
				OtherNonCurrentAssets:                               v.BalanceSheet.OtherNonCurrentAssets,
				DefinedPensionBenefit:                               v.BalanceSheet.DefinedPensionBenefit,
				NonCurrentDeferredTaxesAssets:                       v.BalanceSheet.NonCurrentDeferredTaxesAssets,
				InvestmentinFinancialAssets:                         v.BalanceSheet.InvestmentinFinancialAssets,
				AvailableForSaleSecurities:                          v.BalanceSheet.AvailableForSaleSecurities,
				GoodwillAndOtherIntangibleAssets:                    v.BalanceSheet.GoodwillAndOtherIntangibleAssets,
				OtherIntangibleAssets:                               v.BalanceSheet.OtherIntangibleAssets,
				NetPPE:                                              v.BalanceSheet.NetPPE,
				AccumulatedDepreciation:                             v.BalanceSheet.AccumulatedDepreciation,
				GrossPPE:                                            v.BalanceSheet.GrossPPE,
				ConstructionInProgress:                              v.BalanceSheet.ConstructionInProgress,
				OtherProperties:                                     v.BalanceSheet.OtherProperties,
				MachineryFurnitureEquipment:                         v.BalanceSheet.MachineryFurnitureEquipment,
				BuildingsAndImprovements:                            v.BalanceSheet.BuildingsAndImprovements,
				LandAndImprovements:                                 v.BalanceSheet.LandAndImprovements,
				Properties:                                          v.BalanceSheet.Properties,
				CurrentAssets:                                       v.BalanceSheet.CurrentAssets,
				OtherCurrentAssets:                                  v.BalanceSheet.OtherCurrentAssets,
				HedgingAssetsCurrent:                                v.BalanceSheet.HedgingAssetsCurrent,
				PrepaidAssets:                                       v.BalanceSheet.PrepaidAssets,
				Inventory:                                           v.BalanceSheet.Inventory,
				OtherReceivables:                                    v.BalanceSheet.OtherReceivables,
				AccountsReceivable:                                  v.BalanceSheet.AccountsReceivable,
				CashCashEquivalentsAndShortTermInvestments:          v.BalanceSheet.CashCashEquivalentsAndShortTermInvestments,
				CashAndCashEquivalents:                              v.BalanceSheet.CashAndCashEquivalents,
			},
		})
	}
	var exDividendDate *time.Time
	if bs.Calendar.ExDividendDate != nil {
		var err error
		exDividendDate, err = c.parseBalanceSheetsDateToJST(*bs.Calendar.ExDividendDate)
		if err != nil {
			log.Printf("error parsing time exDividendDate in yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo: %v", err)
			return nil, errors.Wrap(err, "error parsing time exDividendDate in yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo")
		}
	}

	var earningsDate []*time.Time
	for _, v := range bs.Calendar.EarningsDate {
		earningDate, err := c.parseBalanceSheetsDateToJST(*v)
		if err != nil {
			log.Printf("error parsing time earningDate in yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo: %v", err)
			return nil, errors.Wrap(err, "error parsing time earningDate in yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo")
		}
		earningsDate = append(earningsDate, earningDate)
	}

	return &gateway.BalanceSheetsInfo{
		TickerSymbol:  c.symbolSanitizeToStr(bs.Ticker),
		BalanceSheets: bsItem,
		Calendar: &gateway.Calendar{
			ExDividendDate:  exDividendDate,
			EarningsDate:    earningsDate,
			EarningsHigh:    bs.Calendar.EarningsHigh,
			EarningsLow:     bs.Calendar.EarningsLow,
			EarningsAverage: bs.Calendar.EarningsAverage,
			RevenueHigh:     bs.Calendar.RevenueHigh,
			RevenueLow:      bs.Calendar.RevenueLow,
			RevenueAverage:  bs.Calendar.RevenueAverage,
		},
	}, nil
}

func (c *StockAPIClient) getCurrentStockPriceBySymbolYF(ctx context.Context, symbol string) ([]*gateway.StockPrice, error) {
	responseInfo, err := c.getStockPriceChart(
		ctx,
		gateway.StockAPISymbol(c.getYahooFinanceAPIStckBrandSymbol(symbol)),
		gateway.StockAPIInterval1D,
		gateway.StockAPIValidRange1MO,
	)
	if err != nil {
		return nil, errors.Wrap(err, "getCurrentStockPriceBySymbolYF")
	}

	if responseInfo == nil || len(responseInfo.Indicator) == 0 {
		return nil, nil
	}

	stockPrices := make([]*gateway.StockPrice, 0, len(responseInfo.Indicator))
	for _, v := range responseInfo.Indicator {
		stockPrices = append(stockPrices, &gateway.StockPrice{
			Date:            v.Date,
			TickerSymbol:    symbol,
			Open:            v.Open,
			High:            v.High,
			Low:             v.Low,
			Close:           v.Close,
			Volume:          v.Volume,
			AdjustmentClose: v.AdjustmentClose,
		})

	}
	return stockPrices, nil
}

// pythonから来たデータ固有の問題っぽいので、utilには置かない。
func (c *StockAPIClient) parseBalanceSheetsDateToJST(t string) (*time.Time, error) {
	layout := "2006-01-02"
	if strings.Contains(t, "T") {
		layout = "2006-01-02T15:04:05"
	}

	date, err := time.Parse(layout, t)
	if err != nil {
		return nil, err
	}

	// JSTに変換
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return nil, err
	}
	parsedDate := date.In(loc)
	return &parsedDate, nil
}

func (c *StockAPIClient) getYahooFinanceAPIIndexBrandSymbol(symbol string) gateway.StockAPISymbol {
	tickerSymbol := symbol
	if !strings.Contains(symbol, "^") {
		tickerSymbol = fmt.Sprintf("^%s", symbol)
	}
	return gateway.StockAPISymbol(tickerSymbol)
}
func (c *StockAPIClient) getYahooFinanceAPIStckBrandSymbol(symbol string) gateway.StockAPISymbol {
	tickerSymbol := symbol
	if !strings.Contains(symbol, ".T") {
		tickerSymbol = fmt.Sprintf("%s.T", symbol)
	}
	return gateway.StockAPISymbol(tickerSymbol)
}

func (c *StockAPIClient) symbolSanitizeToStr(symbol string) string {
	tickerSymbol := symbol
	if strings.Contains(tickerSymbol, ".T") {
		tickerSymbol = strings.ReplaceAll(tickerSymbol, ".T", "")
	}
	if strings.Contains(tickerSymbol, "^") {
		tickerSymbol = strings.ReplaceAll(tickerSymbol, "^", "")
	}

	return tickerSymbol
}

func (c *StockAPIClient) GetBalanceSheetsBySymbol(ctx context.Context, symbol string) (*gateway.BalanceSheetsInfo, error) {
	// 時価総額は返していない。
	// 発行済み株式数*現在値 で計算したほうが正確であったため。
	cmd := exec.Command(config.YahooFinance().YfinancePyBinaryCMD, fmt.Sprintf("%s.T", symbol))
	output, err := cmd.Output()
	// Notfoundのときは処理続けたい
	if err != nil {
		log.Printf("GetYahooFinanceBalanceSheetsBySymbol error: %v", err)
		return nil, errors.Wrap(err, "YfinancePyBinaryCMD error")
	}

	// JSONデータをマップにパース
	var response YahooFinanceBalanceSheetsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, "YfinancePyBinaryCMD error")
	}

	return c.yahooFinanceBalanceSheetsToYahooFinanceBalanceSheetsInfo(response)
}
