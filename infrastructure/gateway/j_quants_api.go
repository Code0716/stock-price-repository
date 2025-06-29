package gateway

type JQuantsMarketCode TypeOfDocument

const (
	JQuantsMarketCodePrime    JQuantsMarketCode = "0111"
	JQuantsMarketCodeStandard JQuantsMarketCode = "0112"
	JQuantsMarketCodeGrowth   JQuantsMarketCode = "0113"
)

func (j JQuantsMarketCode) String() string {
	return string(j)
}

// TypeOfDocument 決算短信の種類
type TypeOfDocument string

const (
	// 業績予想の修正
	TypeOfDocumentEarnForecastRevision TypeOfDocument = "EarnForecastRevision"
	// 配当予想の修正
	TypeOfDocumentDividendForecastRevision TypeOfDocument = "DividendForecastRevision"

	// 日本基準（JP）
	TypeOfDocumentFYFinancialStatementsConsolidatedJP    TypeOfDocument = "FYFinancialStatements_Consolidated_JP"    // 決算短信（連結・日本基準）
	TypeOfDocumentFYFinancialStatementsNonConsolidatedJP TypeOfDocument = "FYFinancialStatements_NonConsolidated_JP" // 決算短信（非連結・日本基準）
	TypeOfDocument1QFinancialStatementsConsolidatedJP    TypeOfDocument = "1QFinancialStatements_Consolidated_JP"    // 第1四半期決算短信（連結・日本基準）
	TypeOfDocument1QFinancialStatementsNonConsolidatedJP TypeOfDocument = "1QFinancialStatements_NonConsolidated_JP" // 第1四半期決算短信（非連結・日本基準）
	TypeOfDocument2QFinancialStatementsConsolidatedJP    TypeOfDocument = "2QFinancialStatements_Consolidated_JP"    // 第2四半期決算短信（連結・日本基準）
	TypeOfDocument2QFinancialStatementsNonConsolidatedJP TypeOfDocument = "2QFinancialStatements_NonConsolidated_JP" // 第2四半期決算短信（非連結・日本基準）
	TypeOfDocument3QFinancialStatementsConsolidatedJP    TypeOfDocument = "3QFinancialStatements_Consolidated_JP"    // 第3四半期決算短信（連結・日本基準）
	TypeOfDocument3QFinancialStatementsNonConsolidatedJP TypeOfDocument = "3QFinancialStatements_NonConsolidated_JP" // 第3四半期決算短信（非連結・日本基準）

	// 日本版修正国際基準（JMIS）
	TypeOfDocumentFYFinancialStatementsConsolidatedJMIS TypeOfDocument = "FYFinancialStatements_Consolidated_JMIS" // 決算短信（連結・JMIS）
	TypeOfDocument1QFinancialStatementsConsolidatedJMIS TypeOfDocument = "1QFinancialStatements_Consolidated_JMIS" // 第1四半期決算短信（連結・JMIS）
	TypeOfDocument2QFinancialStatementsConsolidatedJMIS TypeOfDocument = "2QFinancialStatements_Consolidated_JMIS" // 第2四半期決算短信（連結・JMIS）
	TypeOfDocument3QFinancialStatementsConsolidatedJMIS TypeOfDocument = "3QFinancialStatements_Consolidated_JMIS" // 第3四半期決算短信（連結・JMIS）

	// 決算短信（連結・米国基準）
	TypeOfDocumentFYFinancialStatementsConsolidatedUS TypeOfDocument = "FYFinancialStatements_Consolidated_US"

	// 国際財務報告基準（IFRS）
	TypeOfDocumentFYFinancialStatementsConsolidatedIFRS    TypeOfDocument = "FYFinancialStatements_Consolidated_IFRS"    // 決算短信（連結・IFRS）
	TypeOfDocumentFYFinancialStatementsNonConsolidatedIFRS TypeOfDocument = "FYFinancialStatements_NonConsolidated_IFRS" // 決算短信（非連結・IFRS）
	TypeOfDocument1QFinancialStatementsConsolidatedIFRS    TypeOfDocument = "1QFinancialStatements_Consolidated_IFRS"    // 第1四半期決算短信（連結・IFRS）
	TypeOfDocument1QFinancialStatementsNonConsolidatedIFRS TypeOfDocument = "1QFinancialStatements_NonConsolidated_IFRS" // 第1四半期決算短信（非連結・IFRS）
	TypeOfDocument2QFinancialStatementsConsolidatedIFRS    TypeOfDocument = "2QFinancialStatements_Consolidated_IFRS"    // 第2四半期決算短信（連結・IFRS）
	TypeOfDocument2QFinancialStatementsNonConsolidatedIFRS TypeOfDocument = "2QFinancialStatements_NonConsolidated_IFRS" // 第2四半期決算短信（非連結・IFRS）
	TypeOfDocument3QFinancialStatementsConsolidatedIFRS    TypeOfDocument = "3QFinancialStatements_Consolidated_IFRS"    // 第3四半期決算短信（連結・IFRS）
	TypeOfDocument3QFinancialStatementsNonConsolidatedIFRS TypeOfDocument = "3QFinancialStatements_NonConsolidated_IFRS" // 第3四半期決算短信（非連結・IFRS）

)

var TypeOfDocumentMap = map[string]TypeOfDocument{
	// 業績予想の修正
	"EarnForecastRevision": TypeOfDocumentEarnForecastRevision,

	// 配当予想の修正
	"DividendForecastRevision": TypeOfDocumentDividendForecastRevision,

	// 日本基準（JP）s
	"FYFinancialStatements_Consolidated_JP":    TypeOfDocumentFYFinancialStatementsConsolidatedJP,
	"FYFinancialStatements_NonConsolidated_JP": TypeOfDocumentFYFinancialStatementsNonConsolidatedJP,
	"1QFinancialStatements_Consolidated_JP":    TypeOfDocument1QFinancialStatementsConsolidatedJP,
	"1QFinancialStatements_NonConsolidated_JP": TypeOfDocument1QFinancialStatementsNonConsolidatedJP,
	"2QFinancialStatements_Consolidated_JP":    TypeOfDocument2QFinancialStatementsConsolidatedJP,
	"2QFinancialStatements_NonConsolidated_JP": TypeOfDocument2QFinancialStatementsNonConsolidatedJP,
	"3QFinancialStatements_Consolidated_JP":    TypeOfDocument3QFinancialStatementsConsolidatedJP,
	"3QFinancialStatements_NonConsolidated_JP": TypeOfDocument3QFinancialStatementsNonConsolidatedJP,

	// 日本版修正国際基準（JMIS）
	"FYFinancialStatements_Consolidated_JMIS": TypeOfDocumentFYFinancialStatementsConsolidatedJMIS,
	"1QFinancialStatements_Consolidated_JMIS": TypeOfDocument1QFinancialStatementsConsolidatedJMIS,
	"2QFinancialStatements_Consolidated_JMIS": TypeOfDocument2QFinancialStatementsConsolidatedJMIS,
	"3QFinancialStatements_Consolidated_JMIS": TypeOfDocument3QFinancialStatementsConsolidatedJMIS,

	// 決算短信（連結・米国基準）
	"FYFinancialStatements_Consolidated_US": TypeOfDocumentFYFinancialStatementsConsolidatedUS,

	// 国際財務報告基準（IFRS）
	"FYFinancialStatements_Consolidated_IFRS":    TypeOfDocumentFYFinancialStatementsConsolidatedIFRS,
	"FYFinancialStatements_NonConsolidated_IFRS": TypeOfDocumentFYFinancialStatementsNonConsolidatedIFRS,
	"1QFinancialStatements_Consolidated_IFRS":    TypeOfDocument1QFinancialStatementsConsolidatedIFRS,
	"1QFinancialStatements_NonConsolidated_IFRS": TypeOfDocument1QFinancialStatementsNonConsolidatedIFRS,
	"2QFinancialStatements_Consolidated_IFRS":    TypeOfDocument2QFinancialStatementsConsolidatedIFRS,
	"2QFinancialStatements_NonConsolidated_IFRS": TypeOfDocument2QFinancialStatementsNonConsolidatedIFRS,
	"3QFinancialStatements_Consolidated_IFRS":    TypeOfDocument3QFinancialStatementsConsolidatedIFRS,
	"3QFinancialStatements_NonConsolidated_IFRS": TypeOfDocument3QFinancialStatementsNonConsolidatedIFRS,
}
