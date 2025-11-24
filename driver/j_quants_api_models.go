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
	Info []struct {
		Date               string `json:"Date"`
		Code               string `json:"Code"`
		CompanyName        string `json:"CompanyName"`
		CompanyNameEnglish string `json:"CompanyNameEnglish"`
		Sector17Code       string `json:"Sector17Code"`
		Sector17CodeName   string `json:"Sector17CodeName"`
		Sector33Code       string `json:"Sector33Code"`
		Sector33CodeName   string `json:"Sector33CodeName"`
		ScaleCategory      string `json:"ScaleCategory"`
		MarketCode         string `json:"MarketCode"`
		MarketCodeName     string `json:"MarketCodeName"`
	} `json:"info"`
}

type jQuantsDailyQuotesResponse struct {
	DailyQuotes []*jQuantsDailyQuote `json:"daily_quotes"`
}

// 日足
type jQuantsDailyQuote struct {
	Date             string          `json:"Date"`
	Code             string          `json:"Code"`
	Open             decimal.Decimal `json:"Open"`
	High             decimal.Decimal `json:"High"`
	Low              decimal.Decimal `json:"Low"`
	Close            decimal.Decimal `json:"Close"`
	UpperLimit       string          `json:"UpperLimit"`
	LowerLimit       string          `json:"LowerLimit"`
	Volume           decimal.Decimal `json:"Volume"`
	TurnoverValue    decimal.Decimal `json:"TurnoverValue"`
	AdjustmentFactor decimal.Decimal `json:"AdjustmentFactor"`
	AdjustmentOpen   decimal.Decimal `json:"AdjustmentOpen"`
	AdjustmentHigh   decimal.Decimal `json:"AdjustmentHigh"`
	AdjustmentLow    decimal.Decimal `json:"AdjustmentLow"`
	AdjustmentClose  decimal.Decimal `json:"AdjustmentClose"`
	AdjustmentVolume decimal.Decimal `json:"AdjustmentVolume"`
}

// 翌営業日に決算発表予定の銘柄
type jQuantsAnnounceFinsScheduleResponse struct {
	Announcement []*AnnounceFinSchedule `json:"announcement"`
}

// 決算予定
type AnnounceFinSchedule struct {
	Date          string `json:"Date"`
	Code          string `json:"Code"`
	CompanyName   string `json:"CompanyName"`
	FiscalYear    string `json:"FiscalYear"`
	SectorName    string `json:"SectorName"`
	FiscalQuarter string `json:"FiscalQuarter"`
	Section       string `json:"Section"`
}

type jQuantsFinancialStatementsResponse struct {
	Statements    []jQuantsFinancialStatement `json:"statements"`
	PaginationKey string                      `json:"pagination_key,omitempty"`
}
type jQuantsFinancialStatement struct {
	DisclosedDate                                                                string `json:"DisclosedDate"`
	DisclosedTime                                                                string `json:"DisclosedTime"`
	LocalCode                                                                    string `json:"LocalCode"`
	DisclosureNumber                                                             string `json:"DisclosureNumber"`
	TypeOfDocument                                                               string `json:"TypeOfDocument"`
	TypeOfCurrentPeriod                                                          string `json:"TypeOfCurrentPeriod"`
	CurrentPeriodStartDate                                                       string `json:"CurrentPeriodStartDate"`
	CurrentPeriodEndDate                                                         string `json:"CurrentPeriodEndDate"`
	CurrentFiscalYearStartDate                                                   string `json:"CurrentFiscalYearStartDate"`
	CurrentFiscalYearEndDate                                                     string `json:"CurrentFiscalYearEndDate"`
	NextFiscalYearStartDate                                                      string `json:"NextFiscalYearStartDate"`
	NextFiscalYearEndDate                                                        string `json:"NextFiscalYearEndDate"`
	NetSales                                                                     string `json:"NetSales"`
	OperatingProfit                                                              string `json:"OperatingProfit"`
	OrdinaryProfit                                                               string `json:"OrdinaryProfit"`
	Profit                                                                       string `json:"Profit"`
	EarningsPerShare                                                             string `json:"EarningsPerShare"`
	DilutedEarningsPerShare                                                      string `json:"DilutedEarningsPerShare"`
	TotalAssets                                                                  string `json:"TotalAssets"`
	Equity                                                                       string `json:"Equity"`
	EquityToAssetRatio                                                           string `json:"EquityToAssetRatio"`
	BookValuePerShare                                                            string `json:"BookValuePerShare"`
	CashFlowsFromOperatingActivities                                             string `json:"CashFlowsFromOperatingActivities"`
	CashFlowsFromInvestingActivities                                             string `json:"CashFlowsFromInvestingActivities"`
	CashFlowsFromFinancingActivities                                             string `json:"CashFlowsFromFinancingActivities"`
	CashAndEquivalents                                                           string `json:"CashAndEquivalents"`
	ResultDividendPerShare1StQuarter                                             string `json:"ResultDividendPerShare1stQuarter"`
	ResultDividendPerShare2NdQuarter                                             string `json:"ResultDividendPerShare2ndQuarter"`
	ResultDividendPerShare3RdQuarter                                             string `json:"ResultDividendPerShare3rdQuarter"`
	ResultDividendPerShareFiscalYearEnd                                          string `json:"ResultDividendPerShareFiscalYearEnd"`
	ResultDividendPerShareAnnual                                                 string `json:"ResultDividendPerShareAnnual"`
	DistributionsPerUnitREIT                                                     string `json:"DistributionsPerUnit(REIT)"`
	ResultTotalDividendPaidAnnual                                                string `json:"ResultTotalDividendPaidAnnual"`
	ResultPayoutRatioAnnual                                                      string `json:"ResultPayoutRatioAnnual"`
	ForecastDividendPerShare1StQuarter                                           string `json:"ForecastDividendPerShare1stQuarter"`
	ForecastDividendPerShare2NdQuarter                                           string `json:"ForecastDividendPerShare2ndQuarter"`
	ForecastDividendPerShare3RdQuarter                                           string `json:"ForecastDividendPerShare3rdQuarter"`
	ForecastDividendPerShareFiscalYearEnd                                        string `json:"ForecastDividendPerShareFiscalYearEnd"`
	ForecastDividendPerShareAnnual                                               string `json:"ForecastDividendPerShareAnnual"`
	ForecastDistributionsPerUnitREIT                                             string `json:"ForecastDistributionsPerUnit(REIT)"`
	ForecastTotalDividendPaidAnnual                                              string `json:"ForecastTotalDividendPaidAnnual"`
	ForecastPayoutRatioAnnual                                                    string `json:"ForecastPayoutRatioAnnual"`
	NextYearForecastDividendPerShare1StQuarter                                   string `json:"NextYearForecastDividendPerShare1stQuarter"`
	NextYearForecastDividendPerShare2NdQuarter                                   string `json:"NextYearForecastDividendPerShare2ndQuarter"`
	NextYearForecastDividendPerShare3RdQuarter                                   string `json:"NextYearForecastDividendPerShare3rdQuarter"`
	NextYearForecastDividendPerShareFiscalYearEnd                                string `json:"NextYearForecastDividendPerShareFiscalYearEnd"`
	NextYearForecastDividendPerShareAnnual                                       string `json:"NextYearForecastDividendPerShareAnnual"`
	NextYearForecastDistributionsPerUnitREIT                                     string `json:"NextYearForecastDistributionsPerUnit(REIT)"`
	NextYearForecastPayoutRatioAnnual                                            string `json:"NextYearForecastPayoutRatioAnnual"`
	ForecastNetSales2NdQuarter                                                   string `json:"ForecastNetSales2ndQuarter"`
	ForecastOperatingProfit2NdQuarter                                            string `json:"ForecastOperatingProfit2ndQuarter"`
	ForecastOrdinaryProfit2NdQuarter                                             string `json:"ForecastOrdinaryProfit2ndQuarter"`
	ForecastProfit2NdQuarter                                                     string `json:"ForecastProfit2ndQuarter"`
	ForecastEarningsPerShare2NdQuarter                                           string `json:"ForecastEarningsPerShare2ndQuarter"`
	NextYearForecastNetSales2NdQuarter                                           string `json:"NextYearForecastNetSales2ndQuarter"`
	NextYearForecastOperatingProfit2NdQuarter                                    string `json:"NextYearForecastOperatingProfit2ndQuarter"`
	NextYearForecastOrdinaryProfit2NdQuarter                                     string `json:"NextYearForecastOrdinaryProfit2ndQuarter"`
	NextYearForecastProfit2NdQuarter                                             string `json:"NextYearForecastProfit2ndQuarter"`
	NextYearForecastEarningsPerShare2NdQuarter                                   string `json:"NextYearForecastEarningsPerShare2ndQuarter"`
	ForecastNetSales                                                             string `json:"ForecastNetSales"`
	ForecastOperatingProfit                                                      string `json:"ForecastOperatingProfit"`
	ForecastOrdinaryProfit                                                       string `json:"ForecastOrdinaryProfit"`
	ForecastProfit                                                               string `json:"ForecastProfit"`
	ForecastEarningsPerShare                                                     string `json:"ForecastEarningsPerShare"`
	NextYearForecastNetSales                                                     string `json:"NextYearForecastNetSales"`
	NextYearForecastOperatingProfit                                              string `json:"NextYearForecastOperatingProfit"`
	NextYearForecastOrdinaryProfit                                               string `json:"NextYearForecastOrdinaryProfit"`
	NextYearForecastProfit                                                       string `json:"NextYearForecastProfit"`
	NextYearForecastEarningsPerShare                                             string `json:"NextYearForecastEarningsPerShare"`
	MaterialChangesInSubsidiaries                                                string `json:"MaterialChangesInSubsidiaries"`
	SignificantChangesInTheScopeOfConsolidation                                  string `json:"SignificantChangesInTheScopeOfConsolidation"`
	ChangesBasedOnRevisionsOfAccountingStandard                                  string `json:"ChangesBasedOnRevisionsOfAccountingStandard"`
	ChangesOtherThanOnesBasedOnRevisionsOfAccountingStandard                     string `json:"ChangesOtherThanOnesBasedOnRevisionsOfAccountingStandard"`
	ChangesInAccountingEstimates                                                 string `json:"ChangesInAccountingEstimates"`
	RetrospectiveRestatement                                                     string `json:"RetrospectiveRestatement"`
	NumberOfIssuedAndOutstandingSharesAtTheEndOfFiscalYearIncludingTreasuryStock string `json:"NumberOfIssuedAndOutstandingSharesAtTheEndOfFiscalYearIncludingTreasuryStock"`
	NumberOfTreasuryStockAtTheEndOfFiscalYear                                    string `json:"NumberOfTreasuryStockAtTheEndOfFiscalYear"`
	AverageNumberOfShares                                                        string `json:"AverageNumberOfShares"`
	NonConsolidatedNetSales                                                      string `json:"NonConsolidatedNetSales"`
	NonConsolidatedOperatingProfit                                               string `json:"NonConsolidatedOperatingProfit"`
	NonConsolidatedOrdinaryProfit                                                string `json:"NonConsolidatedOrdinaryProfit"`
	NonConsolidatedProfit                                                        string `json:"NonConsolidatedProfit"`
	NonConsolidatedEarningsPerShare                                              string `json:"NonConsolidatedEarningsPerShare"`
	NonConsolidatedTotalAssets                                                   string `json:"NonConsolidatedTotalAssets"`
	NonConsolidatedEquity                                                        string `json:"NonConsolidatedEquity"`
	NonConsolidatedEquityToAssetRatio                                            string `json:"NonConsolidatedEquityToAssetRatio"`
	NonConsolidatedBookValuePerShare                                             string `json:"NonConsolidatedBookValuePerShare"`
	ForecastNonConsolidatedNetSales2NdQuarter                                    string `json:"ForecastNonConsolidatedNetSales2ndQuarter"`
	ForecastNonConsolidatedOperatingProfit2NdQuarter                             string `json:"ForecastNonConsolidatedOperatingProfit2ndQuarter"`
	ForecastNonConsolidatedOrdinaryProfit2NdQuarter                              string `json:"ForecastNonConsolidatedOrdinaryProfit2ndQuarter"`
	ForecastNonConsolidatedProfit2NdQuarter                                      string `json:"ForecastNonConsolidatedProfit2ndQuarter"`
	ForecastNonConsolidatedEarningsPerShare2NdQuarter                            string `json:"ForecastNonConsolidatedEarningsPerShare2ndQuarter"`
	NextYearForecastNonConsolidatedNetSales2NdQuarter                            string `json:"NextYearForecastNonConsolidatedNetSales2ndQuarter"`
	NextYearForecastNonConsolidatedOperatingProfit2NdQuarter                     string `json:"NextYearForecastNonConsolidatedOperatingProfit2ndQuarter"`
	NextYearForecastNonConsolidatedOrdinaryProfit2NdQuarter                      string `json:"NextYearForecastNonConsolidatedOrdinaryProfit2ndQuarter"`
	NextYearForecastNonConsolidatedProfit2NdQuarter                              string `json:"NextYearForecastNonConsolidatedProfit2ndQuarter"`
	NextYearForecastNonConsolidatedEarningsPerShare2NdQuarter                    string `json:"NextYearForecastNonConsolidatedEarningsPerShare2ndQuarter"`
	ForecastNonConsolidatedNetSales                                              string `json:"ForecastNonConsolidatedNetSales"`
	ForecastNonConsolidatedOperatingProfit                                       string `json:"ForecastNonConsolidatedOperatingProfit"`
	ForecastNonConsolidatedOrdinaryProfit                                        string `json:"ForecastNonConsolidatedOrdinaryProfit"`
	ForecastNonConsolidatedProfit                                                string `json:"ForecastNonConsolidatedProfit"`
	ForecastNonConsolidatedEarningsPerShare                                      string `json:"ForecastNonConsolidatedEarningsPerShare"`
	NextYearForecastNonConsolidatedNetSales                                      string `json:"NextYearForecastNonConsolidatedNetSales"`
	NextYearForecastNonConsolidatedOperatingProfit                               string `json:"NextYearForecastNonConsolidatedOperatingProfit"`
	NextYearForecastNonConsolidatedOrdinaryProfit                                string `json:"NextYearForecastNonConsolidatedOrdinaryProfit"`
	NextYearForecastNonConsolidatedProfit                                        string `json:"NextYearForecastNonConsolidatedProfit"`
	NextYearForecastNonConsolidatedEarningsPerShare                              string `json:"NextYearForecastNonConsolidatedEarningsPerShare"`
}

// type jQuantsFinancialStatement struct {
// 	FsDetails []struct {
// 		DisclosedDate      string `json:"DisclosedDate"`
// 		DisclosedTime      string `json:"DisclosedTime"`
// 		LocalCode          string `json:"LocalCode"`
// 		DisclosureNumber   string `json:"DisclosureNumber"`
// 		TypeOfDocument     string `json:"TypeOfDocument"`
// 		FinancialStatement struct {
// 			GoodwillIFRS                                                                                             string `json:"Goodwill (IFRS)"`
// 			RetainedEarningsIFRS                                                                                     string `json:"Retained earnings (IFRS)"`
// 			OperatingProfitLossIFRS                                                                                  string `json:"Operating profit (loss) (IFRS)"`
// 			PreviousFiscalYearEndDateDEI                                                                             string `json:"Previous fiscal year end date, DEI"`
// 			BasicEarningsLossPerShareIFRS                                                                            string `json:"Basic earnings (loss) per share (IFRS)"`
// 			DocumentTypeDEI                                                                                          string `json:"Document type, DEI"`
// 			CurrentPeriodEndDateDEI                                                                                  string `json:"Current period end date, DEI"`
// 			Revenue2IFRS                                                                                             string `json:"Revenue - 2 (IFRS)"`
// 			IndustryCodeWhenConsolidatedFinancialStatementsArePreparedInAccordanceWithIndustrySpecificRegulationsDEI string `json:"Industry code when consolidated financial statements are prepared in accordance with industry specific regulations, DEI"`
// 			ProfitLossAttributableToOwnersOfParentIFRS                                                               string `json:"Profit (loss) attributable to owners of parent (IFRS)"`
// 			OtherCurrentLiabilitiesCLIFRS                                                                            string `json:"Other current liabilities - CL (IFRS)"`
// 			ShareOfProfitLossOfInvestmentsAccountedForUsingEquityMethodIFRS                                          string `json:"Share of profit (loss) of investments accounted for using equity method (IFRS)"`
// 			CurrentLiabilitiesIFRS                                                                                   string `json:"Current liabilities (IFRS)"`
// 			EquityAttributableToOwnersOfParentIFRS                                                                   string `json:"Equity attributable to owners of parent (IFRS)"`
// 			WhetherConsolidatedFinancialStatementsArePreparedDEI                                                     string `json:"Whether consolidated financial statements are prepared, DEI"`
// 			NonCurrentLiabilitiesIFRS                                                                                string `json:"Non-current liabilities (IFRS)"`
// 			OtherExpensesIFRS                                                                                        string `json:"Other expenses (IFRS)"`
// 			IncomeTaxesPayableCLIFRS                                                                                 string `json:"Income taxes payable - CL (IFRS)"`
// 			FilerNameInEnglishDEI                                                                                    string `json:"Filer name in English, DEI"`
// 			NonControllingInterestsIFRS                                                                              string `json:"Non-controlling interests (IFRS)"`
// 			CapitalSurplusIFRS                                                                                       string `json:"Capital surplus (IFRS)"`
// 			FinanceCostsIFRS                                                                                         string `json:"Finance costs (IFRS)"`
// 			OtherCurrentAssetsCAIFRS                                                                                 string `json:"Other current assets - CA (IFRS)"`
// 			PropertyPlantAndEquipmentIFRS                                                                            string `json:"Property, plant and equipment (IFRS)"`
// 			DeferredTaxLiabilitiesIFRS                                                                               string `json:"Deferred tax liabilities (IFRS)"`
// 			OtherComponentsOfEquityIFRS                                                                              string `json:"Other components of equity (IFRS)"`
// 			CurrentFiscalYearStartDateDEI                                                                            string `json:"Current fiscal year start date, DEI"`
// 			TypeOfCurrentPeriodDEI                                                                                   string `json:"Type of current period, DEI"`
// 			CashAndCashEquivalentsIFRS                                                                               string `json:"Cash and cash equivalents (IFRS)"`
// 			ShareCapitalIFRS                                                                                         string `json:"Share capital (IFRS)"`
// 			RetirementBenefitAssetNCAIFRS                                                                            string `json:"Retirement benefit asset - NCA (IFRS)"`
// 			NumberOfSubmissionDEI                                                                                    string `json:"Number of submission, DEI"`
// 			TradeAndOtherReceivablesCAIFRS                                                                           string `json:"Trade and other receivables - CA (IFRS)"`
// 			LiabilitiesAndEquityIFRS                                                                                 string `json:"Liabilities and equity (IFRS)"`
// 			EDINETCodeDEI                                                                                            string `json:"EDINET code, DEI"`
// 			EquityIFRS                                                                                               string `json:"Equity (IFRS)"`
// 			SecurityCodeDEI                                                                                          string `json:"Security code, DEI"`
// 			OtherFinancialAssetsCAIFRS                                                                               string `json:"Other financial assets - CA (IFRS)"`
// 			OtherFinancialAssetsNCAIFRS                                                                              string `json:"Other financial assets - NCA (IFRS)"`
// 			IncomeTaxesReceivableCAIFRS                                                                              string `json:"Income taxes receivable - CA (IFRS)"`
// 			InvestmentsAccountedForUsingEquityMethodIFRS                                                             string `json:"Investments accounted for using equity method (IFRS)"`
// 			OtherNonCurrentAssetsNCAIFRS                                                                             string `json:"Other non-current assets - NCA (IFRS)"`
// 			PreviousFiscalYearStartDateDEI                                                                           string `json:"Previous fiscal year start date, DEI"`
// 			FilerNameInJapaneseDEI                                                                                   string `json:"Filer name in Japanese, DEI"`
// 			DeferredTaxAssetsIFRS                                                                                    string `json:"Deferred tax assets (IFRS)"`
// 			TradeAndOtherPayablesCLIFRS                                                                              string `json:"Trade and other payables - CL (IFRS)"`
// 			BondsAndBorrowingsCLIFRS                                                                                 string `json:"Bonds and borrowings - CL (IFRS)"`
// 			CurrentFiscalYearEndDateDEI                                                                              string `json:"Current fiscal year end date, DEI"`
// 			XBRLAmendmentFlagDEI                                                                                     string `json:"XBRL amendment flag, DEI"`
// 			NonCurrentAssetsIFRS                                                                                     string `json:"Non-current assets (IFRS)"`
// 			RetirementBenefitLiabilityNCLIFRS                                                                        string `json:"Retirement benefit liability - NCL (IFRS)"`
// 			AmendmentFlagDEI                                                                                         string `json:"Amendment flag, DEI"`
// 			AssetsIFRS                                                                                               string `json:"Assets (IFRS)"`
// 			IncomeTaxExpenseIFRS                                                                                     string `json:"Income tax expense (IFRS)"`
// 			ReportAmendmentFlagDEI                                                                                   string `json:"Report amendment flag, DEI"`
// 			ProfitLossIFRS                                                                                           string `json:"Profit (loss) (IFRS)"`
// 			OperatingExpensesIFRS                                                                                    string `json:"Operating expenses (IFRS)"`
// 			IntangibleAssetsIFRS                                                                                     string `json:"Intangible assets (IFRS)"`
// 			ProfitLossBeforeTaxFromContinuingOperationsIFRS                                                          string `json:"Profit (loss) before tax from continuing operations (IFRS)"`
// 			LiabilitiesIFRS                                                                                          string `json:"Liabilities (IFRS)"`
// 			AccountingStandardsDEI                                                                                   string `json:"Accounting standards, DEI"`
// 			BondsAndBorrowingsNCLIFRS                                                                                string `json:"Bonds and borrowings - NCL (IFRS)"`
// 			FinanceIncomeIFRS                                                                                        string `json:"Finance income (IFRS)"`
// 			ProfitLossAttributableToNonControllingInterestsIFRS                                                      string `json:"Profit (loss) attributable to non-controlling interests (IFRS)"`
// 			ComparativePeriodEndDateDEI                                                                              string `json:"Comparative period end date, DEI"`
// 			CurrentAssetsIFRS                                                                                        string `json:"Current assets (IFRS)"`
// 			OtherNonCurrentLiabilitiesNCLIFRS                                                                        string `json:"Other non-current liabilities - NCL (IFRS)"`
// 			OtherIncomeIFRS                                                                                          string `json:"Other income (IFRS)"`
// 			TreasurySharesIFRS                                                                                       string `json:"Treasury shares (IFRS)"`
// 		} `json:"FinancialStatement"`
// 	} `json:"fs_details"`
// 	PaginationKey string `json:"pagination_key"`
// }

// TradingCalendarsInfo 相場の営業日の情報のスライス
type TradingCalendarsResponse struct {
	TradingCalendars []TradingCalendar `json:"trading_calendar"`
}

// TradingCalendar 相場の営業日の情報
type TradingCalendar struct {
	Date            string `json:"Date"`
	HolidayDivision string `json:"HolidayDivision"`
}

func (c *StockAPIClient) jQuantsAnnounceFinsScheduleResponseToResponseInfo(
	response jQuantsAnnounceFinsScheduleResponse,
) []*gateway.AnnounceFinScheduleResponseInfo {
	var announcements []*gateway.AnnounceFinScheduleResponseInfo
	for _, v := range response.Announcement {
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
	for _, v := range response.Info {
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
