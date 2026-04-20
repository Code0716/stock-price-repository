//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/util"
)

type jQuantsAPIClientRefreshTokenRequest struct {
	Mailaddress string `json:"mailaddress"`
	Password    string `json:"password"`
}

type jQuantsAPIClientRefreshTokenResponse struct {
	RefreshToken string `json:"refreshToken"`
}

type jQuantsAPIClientIDTokenResponse struct {
	IDToken string `json:"idToken"`
}

type jQuantsStockBrandsResponse struct {
	Data []struct {
		Date               string `json:"Date"`
		Code               string `json:"Code"`
		CompanyName        string `json:"CoName"`
		CompanyNameEnglish string `json:"CoNameEn"`
		Sector17Code       string `json:"S17"`
		Sector17CodeName   string `json:"S17Nm"`
		Sector33Code       string `json:"S33"`
		Sector33CodeName   string `json:"S33Nm"`
		ScaleCategory      string `json:"ScaleCat"`
		MarketCode         string `json:"Mkt"`
		MarketCodeName     string `json:"MktNm"`
	} `json:"data"`
}

type jQuantsDailyQuotesResponse struct {
	DailyQuotes   []*jQuantsDailyQuote `json:"Data"`
	PaginationKey string               `json:"pagination_key"`
}

// 日足
type jQuantsDailyQuote struct {
	Date             string          `json:"Date"`
	Code             string          `json:"Code"`
	Open             decimal.Decimal `json:"O"`
	High             decimal.Decimal `json:"H"`
	Low              decimal.Decimal `json:"L"`
	Close            decimal.Decimal `json:"C"`
	UpperLimit       string          `json:"UL"`
	LowerLimit       string          `json:"LL"`
	Volume           decimal.Decimal `json:"Vo"`
	TurnoverValue    decimal.Decimal `json:"Va"`
	AdjustmentFactor decimal.Decimal `json:"AdjFactor"`
	AdjustmentOpen   decimal.Decimal `json:"AdjO"`
	AdjustmentHigh   decimal.Decimal `json:"AdjH"`
	AdjustmentLow    decimal.Decimal `json:"AdjL"`
	AdjustmentClose  decimal.Decimal `json:"AdjC"`
	AdjustmentVolume decimal.Decimal `json:"AdjVo"`
}

// 翌営業日に決算発表予定の銘柄
type jQuantsAnnounceFinsScheduleResponse struct {
	Data []*AnnounceFinSchedule `json:"data"`
}

// 決算予定
type AnnounceFinSchedule struct {
	Date          string `json:"Date"`
	Code          string `json:"Code"`
	CompanyName   string `json:"CoName"`
	FiscalYear    string `json:"FY"`
	SectorName    string `json:"SectorNm"`
	FiscalQuarter string `json:"FQ"`
	Section       string `json:"Section"`
}

type jQuantsFinancialStatementsResponse struct {
	Statements []jQuantsFinancialStatement `json:"data"`
}

type jQuantsFinancialStatement struct {
	DisclosedDate                                                                string `json:"DiscDate"`
	DisclosedTime                                                                string `json:"DiscTime"`
	LocalCode                                                                    string `json:"Code"`
	DisclosureNumber                                                             string `json:"DiscNo"`
	TypeOfDocument                                                               string `json:"DocType"`
	TypeOfCurrentPeriod                                                          string `json:"CurPerType"`
	CurrentPeriodStartDate                                                       string `json:"CurPerSt"`
	CurrentPeriodEndDate                                                         string `json:"CurPerEn"`
	CurrentFiscalYearStartDate                                                   string `json:"CurFYSt"`
	CurrentFiscalYearEndDate                                                     string `json:"CurFYEn"`
	NextFiscalYearStartDate                                                      string `json:"NxtFYSt"`
	NextFiscalYearEndDate                                                        string `json:"NxtFYEn"`
	NetSales                                                                     string `json:"Sales"`
	OperatingProfit                                                              string `json:"OP"`
	OrdinaryProfit                                                               string `json:"OdP"`
	Profit                                                                       string `json:"NP"`
	EarningsPerShare                                                             string `json:"EPS"`
	DilutedEarningsPerShare                                                      string `json:"DEPS"`
	TotalAssets                                                                  string `json:"TA"`
	Equity                                                                       string `json:"Eq"`
	EquityToAssetRatio                                                           string `json:"EqAR"`
	BookValuePerShare                                                            string `json:"BPS"`
	CashFlowsFromOperatingActivities                                             string `json:"CFO"`
	CashFlowsFromInvestingActivities                                             string `json:"CFI"`
	CashFlowsFromFinancingActivities                                             string `json:"CFF"`
	CashAndEquivalents                                                           string `json:"CashEq"`
	ResultDividendPerShare1StQuarter                                             string `json:"Div1Q"`
	ResultDividendPerShare2NdQuarter                                             string `json:"Div2Q"`
	ResultDividendPerShare3RdQuarter                                             string `json:"Div3Q"`
	ResultDividendPerShareFiscalYearEnd                                          string `json:"DivFY"`
	ResultDividendPerShareAnnual                                                 string `json:"DivAnn"`
	DistributionsPerUnitREIT                                                     string `json:"DivUnit"`
	ResultTotalDividendPaidAnnual                                                string `json:"DivTotalAnn"`
	ResultPayoutRatioAnnual                                                      string `json:"PayoutRatioAnn"`
	ForecastDividendPerShare1StQuarter                                           string `json:"FDiv1Q"`
	ForecastDividendPerShare2NdQuarter                                           string `json:"FDiv2Q"`
	ForecastDividendPerShare3RdQuarter                                           string `json:"FDiv3Q"`
	ForecastDividendPerShareFiscalYearEnd                                        string `json:"FDivFY"`
	ForecastDividendPerShareAnnual                                               string `json:"FDivAnn"`
	ForecastDistributionsPerUnitREIT                                             string `json:"FDivUnit"`
	ForecastTotalDividendPaidAnnual                                              string `json:"FDivTotalAnn"`
	ForecastPayoutRatioAnnual                                                    string `json:"FPayoutRatioAnn"`
	NextYearForecastDividendPerShare1StQuarter                                   string `json:"NxFDiv1Q"`
	NextYearForecastDividendPerShare2NdQuarter                                   string `json:"NxFDiv2Q"`
	NextYearForecastDividendPerShare3RdQuarter                                   string `json:"NxFDiv3Q"`
	NextYearForecastDividendPerShareFiscalYearEnd                                string `json:"NxFDivFY"`
	NextYearForecastDividendPerShareAnnual                                       string `json:"NxFDivAnn"`
	NextYearForecastDistributionsPerUnitREIT                                     string `json:"NxFDivUnit"`
	NextYearForecastPayoutRatioAnnual                                            string `json:"NxFPayoutRatioAnn"`
	ForecastNetSales2NdQuarter                                                   string `json:"FSales2Q"`
	ForecastOperatingProfit2NdQuarter                                            string `json:"FOP2Q"`
	ForecastOrdinaryProfit2NdQuarter                                             string `json:"FOdP2Q"`
	ForecastProfit2NdQuarter                                                     string `json:"FNP2Q"`
	ForecastEarningsPerShare2NdQuarter                                           string `json:"FEPS2Q"`
	NextYearForecastNetSales2NdQuarter                                           string `json:"NxFSales2Q"`
	NextYearForecastOperatingProfit2NdQuarter                                    string `json:"NxFOP2Q"`
	NextYearForecastOrdinaryProfit2NdQuarter                                     string `json:"NxFOdP2Q"`
	NextYearForecastProfit2NdQuarter                                             string `json:"NxFNp2Q"`
	NextYearForecastEarningsPerShare2NdQuarter                                   string `json:"NxFEPS2Q"`
	ForecastNetSales                                                             string `json:"FSales"`
	ForecastOperatingProfit                                                      string `json:"FOP"`
	ForecastOrdinaryProfit                                                       string `json:"FOdP"`
	ForecastProfit                                                               string `json:"FNP"`
	ForecastEarningsPerShare                                                     string `json:"FEPS"`
	NextYearForecastNetSales                                                     string `json:"NxFSales"`
	NextYearForecastOperatingProfit                                              string `json:"NxFOP"`
	NextYearForecastOrdinaryProfit                                               string `json:"NxFOdP"`
	NextYearForecastProfit                                                       string `json:"NxFNp"`
	NextYearForecastEarningsPerShare                                             string `json:"NxFEPS"`
	MaterialChangesInSubsidiaries                                                string `json:"MatChgSub"`
	SignificantChangesInTheScopeOfConsolidation                                  string `json:"SigChgInC"`
	ChangesBasedOnRevisionsOfAccountingStandard                                  string `json:"ChgByASRev"`
	ChangesOtherThanOnesBasedOnRevisionsOfAccountingStandard                     string `json:"ChgNoASRev"`
	ChangesInAccountingEstimates                                                 string `json:"ChgAcEst"`
	RetrospectiveRestatement                                                     string `json:"RetroRst"`
	NumberOfIssuedAndOutstandingSharesAtTheEndOfFiscalYearIncludingTreasuryStock string `json:"ShOutFY"`
	NumberOfTreasuryStockAtTheEndOfFiscalYear                                    string `json:"TrShFY"`
	AverageNumberOfShares                                                        string `json:"AvgSh"`
	NonConsolidatedNetSales                                                      string `json:"NCSales"`
	NonConsolidatedOperatingProfit                                               string `json:"NCOP"`
	NonConsolidatedOrdinaryProfit                                                string `json:"NCOdP"`
	NonConsolidatedProfit                                                        string `json:"NCNP"`
	NonConsolidatedEarningsPerShare                                              string `json:"NCEPS"`
	NonConsolidatedTotalAssets                                                   string `json:"NCTA"`
	NonConsolidatedEquity                                                        string `json:"NCEq"`
	NonConsolidatedEquityToAssetRatio                                            string `json:"NCEqAR"`
	NonConsolidatedBookValuePerShare                                             string `json:"NCBPS"`
	ForecastNonConsolidatedNetSales2NdQuarter                                    string `json:"FNCSales2Q"`
	ForecastNonConsolidatedOperatingProfit2NdQuarter                             string `json:"FNCOP2Q"`
	ForecastNonConsolidatedOrdinaryProfit2NdQuarter                              string `json:"FNCOdP2Q"`
	ForecastNonConsolidatedProfit2NdQuarter                                      string `json:"FNCNP2Q"`
	ForecastNonConsolidatedEarningsPerShare2NdQuarter                            string `json:"FNCEPS2Q"`
	NextYearForecastNonConsolidatedNetSales2NdQuarter                            string `json:"NxFNCSales2Q"`
	NextYearForecastNonConsolidatedOperatingProfit2NdQuarter                     string `json:"NxFNCOP2Q"`
	NextYearForecastNonConsolidatedOrdinaryProfit2NdQuarter                      string `json:"NxFNCOdP2Q"`
	NextYearForecastNonConsolidatedProfit2NdQuarter                              string `json:"NxFNCNP2Q"`
	NextYearForecastNonConsolidatedEarningsPerShare2NdQuarter                    string `json:"NxFNCEPS2Q"`
	ForecastNonConsolidatedNetSales                                              string `json:"FNCSales"`
	ForecastNonConsolidatedOperatingProfit                                       string `json:"FNCOP"`
	ForecastNonConsolidatedOrdinaryProfit                                        string `json:"FNCOdP"`
	ForecastNonConsolidatedProfit                                                string `json:"FNCNP"`
	ForecastNonConsolidatedEarningsPerShare                                      string `json:"FNCEPS"`
	NextYearForecastNonConsolidatedNetSales                                      string `json:"NxFNCSales"`
	NextYearForecastNonConsolidatedOperatingProfit                               string `json:"NxFNCOP"`
	NextYearForecastNonConsolidatedOrdinaryProfit                                string `json:"NxFNCOdP"`
	NextYearForecastNonConsolidatedProfit                                        string `json:"NxFNCNP"`
	NextYearForecastNonConsolidatedEarningsPerShare                              string `json:"NxFNCEPS"`
}

// TradingCalendarsInfo 相場の営業日の情報のスライス
type TradingCalendarsResponse struct {
	TradingCalendars []TradingCalendar `json:"data"`
}

// TradingCalendar 相場の営業日の情報
type TradingCalendar struct {
	Date            string `json:"Date"`
	HolidayDivision string `json:"HolDiv"`
}

func (c *StockAPIClient) jQuantsAnnounceFinsScheduleResponseToResponseInfo(
	response jQuantsAnnounceFinsScheduleResponse,
) []*gateway.AnnounceFinScheduleResponseInfo {
	var announcements []*gateway.AnnounceFinScheduleResponseInfo
	for _, v := range response.Data {
		date, err := util.FormatStringToDate(v.Date)
		if err != nil {
			log.Printf("util.FormatStringToDate: %v", err)
			continue
		}

		announcements = append(announcements, &gateway.AnnounceFinScheduleResponseInfo{
			Date:          date,
			Code:          c.trimSuffixZero(v.Code),
			CompanyName:   v.CompanyName,
			FiscalYear:    v.FiscalYear,
			SectorName:    v.SectorName,
			FiscalQuarter: v.FiscalQuarter,
			Section:       v.Section,
		})
	}
	return announcements
}

func (c *StockAPIClient) jQuantsStockBrandsResponseToResponseInfo(response jQuantsStockBrandsResponse) []*gateway.StockBrand {
	var stockBrands []*gateway.StockBrand
	for _, v := range response.Data {
		date, err := util.FormatStringToDate(v.Date)
		if err != nil {
			log.Printf("util.FormatStringToDate: %v", err)
			continue
		}
		marketCodeName := v.MarketCodeName
		if v.MarketCode == gateway.JQuantsMarketCodePrime.String() ||
			v.MarketCode == gateway.JQuantsMarketCodeStandard.String() ||
			v.MarketCode == gateway.JQuantsMarketCodeGrowth.String() {
			marketCodeName = fmt.Sprintf("%s(国内株式)", v.MarketCodeName)
		}

		marketCode, err := c.trimPrefixZeros(v.MarketCode)
		if err != nil {
			log.Printf("strconv.trimPrefixZeros:%v", err)
			continue
		}

		stockBrands = append(stockBrands, &gateway.StockBrand{
			Date:             date,
			Symbol:           c.trimSuffixZero(v.Code),
			CompanyName:      v.CompanyName,
			Sector33Code:     v.Sector33Code,
			Sector33CodeName: v.Sector33CodeName,
			Sector17Code:     v.Sector17Code,
			Sector17CodeName: v.Sector17CodeName,
			ScaleCategory:    v.ScaleCategory,
			MarketCode:       marketCode,
			MarketCodeName:   marketCodeName,
		})
	}
	return stockBrands
}

func (c *StockAPIClient) jQuantsDailyQuotesResponseToResponseInfo(response jQuantsDailyQuotesResponse) []*gateway.StockPrice {
	if len(response.DailyQuotes) == 0 {
		return nil
	}

	var responseInfo []*gateway.StockPrice
	for _, v := range response.DailyQuotes {
		date, err := util.FormatStringToDate(v.Date)
		if err != nil {
			log.Printf("jQuantsDailyQuotesResponseToResponseInfo error: %v", err)
			continue
		}

		tickerSymbol := c.trimSuffixZero(v.Code)

		responseInfo = append(responseInfo, &gateway.StockPrice{
			Date:             date,
			TickerSymbol:     tickerSymbol,
			Open:             v.Open,
			High:             v.High,
			Low:              v.Low,
			Close:            v.Close,
			UpperLimit:       v.UpperLimit,
			LowerLimit:       v.LowerLimit,
			Volume:           v.Volume.IntPart(),
			TurnoverValue:    v.TurnoverValue,
			AdjustmentFactor: v.AdjustmentFactor,
			AdjustmentOpen:   v.AdjustmentOpen,
			AdjustmentHigh:   v.AdjustmentHigh,
			AdjustmentLow:    v.AdjustmentLow,
			AdjustmentClose:  v.AdjustmentClose,
			AdjustmentVolume: v.AdjustmentVolume,
		})
	}
	return responseInfo
}

// 証券コードの末尾の0をトリムする。
func (c *StockAPIClient) trimSuffixZero(s string) string {
	if strings.HasSuffix(s, "0") {
		return s[:len(s)-1]
	}
	return s
}

func (c *StockAPIClient) trimPrefixZeros(s string) (string, error) {
	strInt, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(strInt, 10), nil
}

func (c *StockAPIClient) jQuantsFinancialStatementsToGatewayModels(response jQuantsFinancialStatementsResponse) []*gateway.FinancialStatementsResponseInfo {
	if len(response.Statements) == 0 {
		return nil
	}
	responseInfo := make([]*gateway.FinancialStatementsResponseInfo, 0, len(response.Statements))
	for _, v := range response.Statements {
		responseInfo = append(responseInfo, c.jQuantsFinancialStatementToGatewayModel(v))
	}
	return responseInfo
}

func (c *StockAPIClient) jQuantsFinancialStatementToGatewayModel(statement jQuantsFinancialStatement) *gateway.FinancialStatementsResponseInfo {
	return &gateway.FinancialStatementsResponseInfo{
		DisclosedDate:                                 statement.DisclosedDate,
		CurrentPeriodEndDate:                          statement.CurrentPeriodEndDate,
		TickerSymbol:                                  c.trimSuffixZero(statement.LocalCode),
		TypeOfDocument:                                statement.TypeOfDocument,
		TypeOfCurrentPeriod:                           statement.TypeOfCurrentPeriod,
		ForecastDividendPerShareFiscalYearEnd:         statement.ForecastDividendPerShareFiscalYearEnd,
		ForecastDividendPerShareAnnual:                statement.ForecastDividendPerShareAnnual,
		NextYearForecastDividendPerShareFiscalYearEnd: statement.NextYearForecastDividendPerShareFiscalYearEnd,
		NextYearForecastDividendPerShareAnnual:        statement.NextYearForecastDividendPerShareAnnual,
		NextFiscalYearEndDate:                         statement.NextFiscalYearEndDate,
	}
}

func (c *StockAPIClient) jQuantsTradingCalendarsToGatewayModels(response TradingCalendarsResponse) []*gateway.TradingCalendarsInfo {
	if len(response.TradingCalendars) == 0 {
		return nil
	}
	responseInfo := make([]*gateway.TradingCalendarsInfo, 0, len(response.TradingCalendars))
	for _, v := range response.TradingCalendars {
		date, err := util.FormatStringToDate(v.Date)
		if err != nil {
			log.Printf("util.FormatStringToDate: %v", err)
			continue
		}
		responseInfo = append(responseInfo, &gateway.TradingCalendarsInfo{
			Date:            date,
			HolidayDivision: v.HolidayDivision,
		})
	}
	return responseInfo
}
