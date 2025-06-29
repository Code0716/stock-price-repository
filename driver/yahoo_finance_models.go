//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"github.com/shopspring/decimal"
)

type yahooFinanceAPIResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency             string          `json:"currency"`
				Symbol               string          `json:"symbol"`
				ExchangeName         string          `json:"exchangeName"`
				FullExchangeName     string          `json:"fullExchangeName"`
				InstrumentType       string          `json:"instrumentType"`
				FirstTradeDate       int64           `json:"firstTradeDate"`
				RegularMarketTime    int64           `json:"regularMarketTime"`
				HasPrePostMarketData bool            `json:"hasPrePostMarketData"`
				Gmtoffset            int64           `json:"gmtoffset"`
				Timezone             string          `json:"timezone"`
				ExchangeTimezoneName string          `json:"exchangeTimezoneName"`
				RegularMarketPrice   decimal.Decimal `json:"regularMarketPrice"`
				FiftyTwoWeekHigh     decimal.Decimal `json:"fiftyTwoWeekHigh"`
				FiftyTwoWeekLow      decimal.Decimal `json:"fiftyTwoWeekLow"`
				RegularMarketDayHigh decimal.Decimal `json:"regularMarketDayHigh"`
				RegularMarketDayLow  decimal.Decimal `json:"regularMarketDayLow"`
				RegularMarketVolume  int64           `json:"regularMarketVolume"`
				LongName             string          `json:"longName"`
				ShortName            string          `json:"shortName"`
				ChartPreviousClose   decimal.Decimal `json:"chartPreviousClose"`
				PriceHint64          int64           `json:"priceHint"`
				CurrentTradingPeriod struct {
					Pre struct {
						Timezone  string `json:"timezone"`
						Start     int64  `json:"start"`
						End       int64  `json:"end"`
						Gmtoffset int64  `json:"gmtoffset"`
					} `json:"pre"`
					Regular struct {
						Timezone  string `json:"timezone"`
						Start     int64  `json:"start"`
						End       int64  `json:"end"`
						Gmtoffset int64  `json:"gmtoffset"`
					} `json:"regular"`
					Post struct {
						Timezone  string `json:"timezone"`
						Start     int64  `json:"start"`
						End       int64  `json:"end"`
						Gmtoffset int64  `json:"gmtoffset"`
					} `json:"post"`
				} `json:"currentTradingPeriod"`
				DataGranularity string   `json:"dataGranularity"`
				Range           string   `json:"range"`
				ValidRanges     []string `json:"validRanges"`
			} `json:"meta"`
			Timestamp  []int64                `json:"timestamp"`
			Indicators YahooFinanceIndicators `json:"indicators"`
		} `json:"result"`
		Error struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

type YahooFinanceIndicators struct {
	Quote    []YahooFinanceQuote    `json:"quote"`
	Adjclose []YahooFinanceAdjclose `json:"adjclose"`
}

type YahooFinanceQuote struct {
	Close  []decimal.Decimal `json:"close"`
	Low    []decimal.Decimal `json:"low"`
	High   []decimal.Decimal `json:"high"`
	Volume []int64           `json:"volume"`
	Open   []decimal.Decimal `json:"open"`
}
type YahooFinanceAdjclose struct {
	Adjclose []decimal.Decimal `json:"adjclose"`
}

type YahooFinanceBalanceSheetsResponse struct {
	Ticker        string `json:"ticker"`
	BalanceSheets []struct {
		Date         string `json:"date"`
		BalanceSheet struct {
			TreasurySharesNumber                                decimal.Decimal `json:"Treasury Shares Number"`
			OrdinarySharesNumber                                decimal.Decimal `json:"Ordinary Shares Number"`
			ShareIssued                                         decimal.Decimal `json:"Share Issued"`
			TotalDebt                                           decimal.Decimal `json:"Total Debt"`
			TangibleBookValue                                   decimal.Decimal `json:"Tangible Book Value"`
			InvestedCapital                                     decimal.Decimal `json:"Invested Capital"`
			WorkingCapital                                      decimal.Decimal `json:"Working Capital"`
			NetTangibleAssets                                   decimal.Decimal `json:"Net Tangible Assets"`
			CapitalLeaseObligations                             decimal.Decimal `json:"Capital Lease Obligations"`
			CommonStockEquity                                   decimal.Decimal `json:"Common Stock Equity"`
			TotalCapitalization                                 decimal.Decimal `json:"Total Capitalization"`
			TotalEquityGrossMinorityInterest                    decimal.Decimal `json:"Total Equity Gross Minority Interest"`
			MinorityInterest                                    decimal.Decimal `json:"Minority Interest"`
			StockholdersEquity                                  decimal.Decimal `json:"Stockholders Equity"`
			TreasuryStock                                       decimal.Decimal `json:"Treasury Stock"`
			RetainedEarnings                                    decimal.Decimal `json:"Retained Earnings"`
			AdditionalPaidInCapital                             decimal.Decimal `json:"Additional Paid In Capital"`
			CapitalStock                                        decimal.Decimal `json:"Capital Stock"`
			CommonStock                                         decimal.Decimal `json:"Common Stock"`
			TotalLiabilitiesNetMinorityInterest                 decimal.Decimal `json:"Total Liabilities Net Minority Interest"`
			TotalNonCurrentLiabilitiesNetMinorityInterest       decimal.Decimal `json:"Total Non Current Liabilities Net Minority Interest"`
			OtherNonCurrentLiabilities                          decimal.Decimal `json:"Other Non Current Liabilities"`
			NonCurrentPensionAndOtherPostretirementBenefitPlans decimal.Decimal `json:"Non Current Pension And Other Postretirement Benefit Plans"`
			NonCurrentDeferredTaxesLiabilities                  decimal.Decimal `json:"Non Current Deferred Taxes Liabilities"`
			LongTermDebtAndCapitalLeaseObligation               decimal.Decimal `json:"Long Term Debt And Capital Lease Obligation"`
			LongTermCapitalLeaseObligation                      decimal.Decimal `json:"Long Term Capital Lease Obligation"`
			LongTermDebt                                        decimal.Decimal `json:"Long Term Debt"`
			LongTermProvisions                                  decimal.Decimal `json:"Long Term Provisions"`
			CurrentLiabilities                                  decimal.Decimal `json:"Current Liabilities"`
			OtherCurrentLiabilities                             decimal.Decimal `json:"Other Current Liabilities"`
			CurrentDebtAndCapitalLeaseObligation                decimal.Decimal `json:"Current Debt And Capital Lease Obligation"`
			CurrentDebt                                         decimal.Decimal `json:"Current Debt"`
			PensionandOtherPostRetirementBenefitPlansCurrent    decimal.Decimal `json:"Pensionand Other Post Retirement Benefit Plans Current"`
			CurrentProvisions                                   decimal.Decimal `json:"Current Provisions"`
			Payables                                            decimal.Decimal `json:"Payables"`
			TotalTaxPayable                                     decimal.Decimal `json:"Total Tax Payable"`
			AccountsPayable                                     decimal.Decimal `json:"Accounts Payable"`
			TotalAssets                                         decimal.Decimal `json:"Total Assets"`
			TotalNonCurrentAssets                               decimal.Decimal `json:"Total Non Current Assets"`
			OtherNonCurrentAssets                               decimal.Decimal `json:"Other Non Current Assets"`
			DefinedPensionBenefit                               decimal.Decimal `json:"Defined Pension Benefit"`
			NonCurrentDeferredTaxesAssets                       decimal.Decimal `json:"Non Current Deferred Taxes Assets"`
			InvestmentinFinancialAssets                         decimal.Decimal `json:"Investmentin Financial Assets"`
			AvailableForSaleSecurities                          decimal.Decimal `json:"Available For Sale Securities"`
			GoodwillAndOtherIntangibleAssets                    decimal.Decimal `json:"Goodwill And Other Intangible Assets"`
			OtherIntangibleAssets                               decimal.Decimal `json:"Other Intangible Assets"`
			NetPPE                                              decimal.Decimal `json:"Net PPE"`
			AccumulatedDepreciation                             decimal.Decimal `json:"Accumulated Depreciation"`
			GrossPPE                                            decimal.Decimal `json:"Gross PPE"`
			ConstructionInProgress                              decimal.Decimal `json:"Construction In Progress"`
			OtherProperties                                     decimal.Decimal `json:"Other Properties"`
			MachineryFurnitureEquipment                         decimal.Decimal `json:"Machinery Furniture Equipment"`
			BuildingsAndImprovements                            decimal.Decimal `json:"Buildings And Improvements"`
			LandAndImprovements                                 decimal.Decimal `json:"Land And Improvements"`
			Properties                                          decimal.Decimal `json:"Properties"`
			CurrentAssets                                       decimal.Decimal `json:"Current Assets"`
			OtherCurrentAssets                                  decimal.Decimal `json:"Other Current Assets"`
			HedgingAssetsCurrent                                decimal.Decimal `json:"Hedging Assets Current"`
			PrepaidAssets                                       decimal.Decimal `json:"Prepaid Assets"`
			Inventory                                           decimal.Decimal `json:"Inventory"`
			OtherReceivables                                    decimal.Decimal `json:"Other Receivables"`
			AccountsReceivable                                  decimal.Decimal `json:"Accounts Receivable"`
			CashCashEquivalentsAndShortTermInvestments          decimal.Decimal `json:"Cash Cash Equivalents And Short Term Investments"`
			CashAndCashEquivalents                              decimal.Decimal `json:"Cash And Cash Equivalents"`
		} `json:"balance_sheet"`
	} `json:"balance_sheets"`
	Calendar struct {
		ExDividendDate  *string         `json:"Ex-Dividend Date"`
		EarningsDate    []*string       `json:"Earnings Date"`
		EarningsHigh    decimal.Decimal `json:"Earnings High"`
		EarningsLow     decimal.Decimal `json:"Earnings Low"`
		EarningsAverage decimal.Decimal `json:"Earnings Average"`
		RevenueHigh     int64           `json:"Revenue High"`
		RevenueLow      int64           `json:"Revenue Low"`
		RevenueAverage  int64           `json:"Revenue Average"`
	} `json:"calendar"`
}
