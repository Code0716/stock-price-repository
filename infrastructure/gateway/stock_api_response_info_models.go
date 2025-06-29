package gateway

import (
	"time"

	"github.com/shopspring/decimal"
)

type StockBrand struct {
	Date             time.Time
	Symbol           string
	CompanyName      string
	Sector33Code     string
	Sector33CodeName string
	Sector17Code     string
	Sector17CodeName string
	ScaleCategory    string
	MarketCode       string
	MarketCodeName   string
}

// TODO: 不要なプロパティは削る
type StockPrice struct {
	Date             time.Time
	TickerSymbol     string
	Open             decimal.Decimal
	High             decimal.Decimal
	Low              decimal.Decimal
	Close            decimal.Decimal
	UpperLimit       string
	LowerLimit       string
	Volume           int64
	TurnoverValue    decimal.Decimal
	AdjustmentFactor decimal.Decimal
	AdjustmentOpen   decimal.Decimal
	AdjustmentHigh   decimal.Decimal
	AdjustmentLow    decimal.Decimal
	AdjustmentClose  decimal.Decimal
	AdjustmentVolume decimal.Decimal
}

type BalanceSheet struct {
	TreasurySharesNumber                                decimal.Decimal
	OrdinarySharesNumber                                decimal.Decimal
	ShareIssued                                         decimal.Decimal // 発行済株式数
	TotalDebt                                           decimal.Decimal
	TangibleBookValue                                   decimal.Decimal
	InvestedCapital                                     decimal.Decimal
	WorkingCapital                                      decimal.Decimal
	NetTangibleAssets                                   decimal.Decimal
	CapitalLeaseObligations                             decimal.Decimal
	CommonStockEquity                                   decimal.Decimal
	TotalCapitalization                                 decimal.Decimal
	TotalEquityGrossMinorityInterest                    decimal.Decimal
	MinorityInterest                                    decimal.Decimal
	StockholdersEquity                                  decimal.Decimal
	TreasuryStock                                       decimal.Decimal
	RetainedEarnings                                    decimal.Decimal
	AdditionalPaidInCapital                             decimal.Decimal
	CapitalStock                                        decimal.Decimal
	CommonStock                                         decimal.Decimal
	TotalLiabilitiesNetMinorityInterest                 decimal.Decimal // 負債合計
	TotalNonCurrentLiabilitiesNetMinorityInterest       decimal.Decimal
	OtherNonCurrentLiabilities                          decimal.Decimal
	NonCurrentPensionAndOtherPostretirementBenefitPlans decimal.Decimal
	NonCurrentDeferredTaxesLiabilities                  decimal.Decimal
	LongTermDebtAndCapitalLeaseObligation               decimal.Decimal
	LongTermCapitalLeaseObligation                      decimal.Decimal
	LongTermDebt                                        decimal.Decimal
	LongTermProvisions                                  decimal.Decimal
	CurrentLiabilities                                  decimal.Decimal
	OtherCurrentLiabilities                             decimal.Decimal
	CurrentDebtAndCapitalLeaseObligation                decimal.Decimal
	CurrentDebt                                         decimal.Decimal
	PensionandOtherPostRetirementBenefitPlansCurrent    decimal.Decimal
	CurrentProvisions                                   decimal.Decimal
	Payables                                            decimal.Decimal
	TotalTaxPayable                                     decimal.Decimal
	AccountsPayable                                     decimal.Decimal
	TotalAssets                                         decimal.Decimal // 銀行とかCurrent Assetsがない場合はこっちを確認するべきか。そもそもCurrent Assetsがなけれ計算しないでいいかも
	TotalNonCurrentAssets                               decimal.Decimal
	OtherNonCurrentAssets                               decimal.Decimal
	DefinedPensionBenefit                               decimal.Decimal
	NonCurrentDeferredTaxesAssets                       decimal.Decimal
	InvestmentinFinancialAssets                         decimal.Decimal // 金融資産への投資
	AvailableForSaleSecurities                          decimal.Decimal // 売却可能有価証券 InvestmentinFinancialAssetsと同じ金額になることが多いようだが、売却可能な方を使ったほうが正確であろう。
	GoodwillAndOtherIntangibleAssets                    decimal.Decimal
	OtherIntangibleAssets                               decimal.Decimal
	NetPPE                                              decimal.Decimal
	AccumulatedDepreciation                             decimal.Decimal
	GrossPPE                                            decimal.Decimal
	ConstructionInProgress                              decimal.Decimal
	OtherProperties                                     decimal.Decimal
	MachineryFurnitureEquipment                         decimal.Decimal
	BuildingsAndImprovements                            decimal.Decimal
	LandAndImprovements                                 decimal.Decimal
	Properties                                          decimal.Decimal
	CurrentAssets                                       decimal.Decimal
	OtherCurrentAssets                                  decimal.Decimal
	HedgingAssetsCurrent                                decimal.Decimal
	PrepaidAssets                                       decimal.Decimal
	Inventory                                           decimal.Decimal
	OtherReceivables                                    decimal.Decimal
	AccountsReceivable                                  decimal.Decimal
	CashCashEquivalentsAndShortTermInvestments          decimal.Decimal
	CashAndCashEquivalents                              decimal.Decimal
}

type BalanceSheetItem struct {
	Date         time.Time
	BalanceSheet BalanceSheet
}

// 決算予定
type AnnounceFinScheduleResponseInfo struct {
	Date          time.Time
	Code          string
	CompanyName   string
	FiscalYear    string
	SectorName    string
	FiscalQuarter string
	Section       string
}

// J-Quants APIから取得した決算短信の情報。必要なところのみ。
// 別途必要になっているなら、このモデルを拡張する。
type FinancialStatementsResponseInfo struct {
	DisclosedDate                                 string
	TickerSymbol                                  string
	CurrentPeriodEndDate                          string // 当期末日
	TypeOfDocument                                string // 開示書類種別
	TypeOfCurrentPeriod                           string // 当会計期間の種類
	ForecastDividendPerShareFiscalYearEnd         string // 1株あたり当期末予想配当
	ForecastDividendPerShareAnnual                string // 1株あたり当期予想配当の合計
	NextYearForecastDividendPerShareFiscalYearEnd string // 一株あたり配当予想 翌事業年度期末
	NextYearForecastDividendPerShareAnnual        string // 一株あたり配当予想 翌事業年度合計
	NextFiscalYearEndDate                         string // 翌事業年度末日
}
