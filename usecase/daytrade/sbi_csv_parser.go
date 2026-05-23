package daytrade

import (
	"bytes"
	"encoding/csv"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/Code0716/stock-price-repository/models"
)

const (
	dataColumns      = 9
	maxCSVBytes      = 10 << 20 // 10MB
	executedOnLayout = "2006/1/2"
)

// ErrParse CSV パースエラーの sentinel
var ErrParse = errors.New("csv parse error")

// columnMap ヘッダ行から検出した各カラムの列インデックス。
// tradeKindCol == -1 は新フォーマット (tradeKind は空文字固定)。
type columnMap struct {
	brand       int
	tradeKindCol int // -1 = 新フォーマット (取引 列が marginKind を意味する)
	marginKind  int
	quantity    int
	tradeAmount int
	unitPrice   int
	averageCost int
	profitLoss  int
}

// defaultColumnMap 旧フォーマット固定位置フォールバック。
// ヘッダ行が検出できなかった場合に使用 (後方互換)。
var defaultColumnMap = columnMap{
	brand:        2,
	tradeKindCol: 1,
	marginKind:   3,
	quantity:     4,
	tradeAmount:  5,
	unitPrice:    6,
	averageCost:  7,
	profitLoss:   8,
}

// detectColumnMap ヘッダ行 (「約定日」を含む行) から列名→インデックスの map を構築する。
// 旧フォーマット: 「信用」列あり → tradeKind=取引列, marginKind=信用列
// 新フォーマット: 「口座」列あり / 「信用」列なし → tradeKind="", marginKind=取引列
func detectColumnMap(record []string) (columnMap, bool) {
	idx := make(map[string]int, len(record))
	for i, v := range record {
		idx[strings.TrimSpace(v)] = i
	}

	// 必須: 約定日
	if _, ok := idx["約定日"]; !ok {
		return columnMap{}, false
	}

	cm := columnMap{}

	// 銘柄列: 旧=「銘柄」/ 新=「銘柄名」
	if i, ok := idx["銘柄名"]; ok {
		cm.brand = i
	} else if i, ok := idx["銘柄"]; ok {
		cm.brand = i
	} else {
		return columnMap{}, false
	}

	// 数量
	i, ok := idx["数量"]
	if !ok {
		return columnMap{}, false
	}
	cm.quantity = i

	// 約定代金: 旧=「約定代金」/ 新=「売却/決済額」
	if i, ok = idx["売却/決済額"]; ok {
		cm.tradeAmount = i
	} else if i, ok = idx["約定代金"]; ok {
		cm.tradeAmount = i
	} else {
		return columnMap{}, false
	}

	// 単価
	if i, ok = idx["単価"]; !ok {
		return columnMap{}, false
	}
	cm.unitPrice = i

	// 平均取得単価: 旧=「平均取得単価」/ 新=「平均取得価額」
	if i, ok = idx["平均取得価額"]; ok {
		cm.averageCost = i
	} else if i, ok = idx["平均取得単価"]; ok {
		cm.averageCost = i
	} else {
		return columnMap{}, false
	}

	// 損益: 旧=「売買損益(税引前・円)」/ 新=「実現損益(税引前・円)」
	foundPL := false
	for _, name := range record {
		n := strings.TrimSpace(name)
		if strings.Contains(n, "損益") {
			cm.profitLoss = idx[n]
			foundPL = true
			break
		}
	}
	if !foundPL {
		return columnMap{}, false
	}

	// フォーマット判別: 「信用」列があれば旧、なければ新
	if i, ok = idx["信用"]; ok {
		// 旧フォーマット
		cm.tradeKindCol = idx["取引"]
		cm.marginKind = i
	} else {
		// 新フォーマット: 「取引」列が marginKind に相当
		cm.tradeKindCol = -1
		if i, ok = idx["取引"]; !ok {
			return columnMap{}, false
		}
		cm.marginKind = i
	}

	return cm, true
}

// naturalKey は occurrence_no 採番に用いる同一性判定キー
type naturalKey struct {
	executedOn  string
	tickerSymbol string
	tradeKind   string
	marginKind  string
	quantity    string
	tradeAmount string
	unitPrice   string
	profitLoss  string
}

// ParseSBIDaytradeCSV SBI証券の国内株式取引履歴CSVをパースしてドメインモデルに変換する。
// CSVはShift-JIS(CP932)で出力される。UTF-8 BOMがあればUTF-8として扱う。
// ヘッダ行 (「約定日」を含む行) を動的に検出し、旧・新フォーマット両対応。
// 同一自然キーの行が複数ある場合は occurrence_no (0始まり) で区別する。
func ParseSBIDaytradeCSV(r io.Reader, now time.Time) ([]*models.DaytradeExecution, error) {
	raw, err := io.ReadAll(io.LimitReader(r, int64(maxCSVBytes)+1))
	if err != nil {
		return nil, errors.Wrap(wrapParseError(err), "read csv body")
	}
	if len(raw) > maxCSVBytes {
		return nil, wrapParseError(errors.New("csv too large (>10MB)"))
	}

	decoded, err := decodeBytes(raw)
	if err != nil {
		return nil, wrapParseError(err)
	}

	reader := csv.NewReader(bytes.NewReader(decoded))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	cm := defaultColumnMap
	cmDetected := false

	var executions []*models.DaytradeExecution
	recNo := 0
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, wrapParseError(errors.Wrapf(err, "csv parse record %d", recNo+1))
		}
		recNo++

		// ヘッダ行の検出 (1回だけ)
		if !cmDetected && len(record) > 0 && strings.TrimSpace(record[0]) == "約定日" {
			if detected, ok := detectColumnMap(record); ok {
				cm = detected
				cmDetected = true
			}
			continue
		}

		if len(record) < dataColumns {
			continue
		}
		// データ行は先頭列が "YYYY/M/D" 形式の日付
		if _, dateErr := time.ParseInLocation(executedOnLayout, strings.TrimSpace(record[0]), time.Local); dateErr != nil {
			continue
		}
		ex, err := buildExecutionMapped(record, cm, now)
		if err != nil {
			return nil, wrapParseError(errors.Wrapf(err, "csv data record %d", recNo))
		}
		executions = append(executions, ex)
	}

	// occurrence_no 採番: 同一自然キーのグループ内で 0, 1, 2... を割り振る
	assignOccurrenceNo(executions)

	return executions, nil
}

// assignOccurrenceNo は slice を走査し、同一自然キーの行に 0始まりの通し番号を付与する。
// CSV の出現順を保持するため slice を前から舐める。
func assignOccurrenceNo(rows []*models.DaytradeExecution) {
	counts := make(map[naturalKey]uint32)
	for _, ex := range rows {
		k := naturalKey{
			executedOn:   ex.ExecutedOn.Format("2006-01-02"),
			tickerSymbol: ex.TickerSymbol,
			tradeKind:    ex.TradeKind,
			marginKind:   ex.MarginKind,
			quantity:     strconv.FormatUint(uint64(ex.Quantity), 10),
			tradeAmount:  strconv.FormatInt(ex.TradeAmount, 10),
			unitPrice:    ex.UnitPrice.String(),
			profitLoss:   strconv.FormatInt(ex.ProfitLoss, 10),
		}
		ex.OccurrenceNo = counts[k]
		counts[k]++
	}
}

func wrapParseError(err error) error {
	return errors.WithMessage(ErrParse, err.Error())
}

// decodeBytes バイト列をUTF-8に変換する。
// 優先順位: UTF-16 BOM拒否 → UTF-8 BOM剥がし → Shift-JIS → UTF-8フォールバック
func decodeBytes(b []byte) ([]byte, error) {
	if bytes.HasPrefix(b, []byte{0xFF, 0xFE}) || bytes.HasPrefix(b, []byte{0xFE, 0xFF}) {
		return nil, errors.New("unsupported encoding: UTF-16")
	}
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		return b[3:], nil
	}
	// Shift-JIS (CP932) として decode
	sjisOut, sjisErr := io.ReadAll(transform.NewReader(bytes.NewReader(b), japanese.ShiftJIS.NewDecoder()))
	if sjisErr == nil {
		return sjisOut, nil
	}
	// Shift-JIS 失敗時はUTF-8として妥当性チェック
	if utf8.Valid(b) {
		return b, nil
	}
	return nil, errors.Wrap(sjisErr, "failed to decode CSV (tried Shift-JIS and UTF-8)")
}

var brandRe = regexp.MustCompile(`^(.+?)[\s　]+([0-9A-Z]{4,5})$`)

// buildExecutionMapped は columnMap を使ってデータ行をドメインモデルに変換する。
func buildExecutionMapped(record []string, cm columnMap, now time.Time) (*models.DaytradeExecution, error) {
	executedOn, err := time.ParseInLocation("2006/1/2", strings.TrimSpace(record[0]), time.Local)
	if err != nil {
		return nil, errors.Wrap(err, "executedOn")
	}

	var tradeKind string
	if cm.tradeKindCol >= 0 {
		tradeKind = strings.TrimSpace(record[cm.tradeKindCol])
	}

	brandName, tickerSymbol, err := splitBrand(record[cm.brand])
	if err != nil {
		return nil, err
	}

	marginKind := strings.TrimSpace(record[cm.marginKind])

	quantity, err := parseUintComma(record[cm.quantity])
	if err != nil {
		return nil, errors.Wrap(err, "quantity")
	}

	tradeAmount, err := parseIntComma(record[cm.tradeAmount])
	if err != nil {
		return nil, errors.Wrap(err, "tradeAmount")
	}

	unitPrice, err := parseDecimalComma(record[cm.unitPrice])
	if err != nil {
		return nil, errors.Wrap(err, "unitPrice")
	}

	averageCost, err := parseDecimalComma(record[cm.averageCost])
	if err != nil {
		return nil, errors.Wrap(err, "averageCost")
	}

	profitLoss, err := parseSignedIntComma(record[cm.profitLoss])
	if err != nil {
		return nil, errors.Wrap(err, "profitLoss")
	}

	return &models.DaytradeExecution{
		ExecutedOn:   executedOn,
		TradeKind:    tradeKind,
		MarginKind:   marginKind,
		TickerSymbol: tickerSymbol,
		BrandName:    brandName,
		Quantity:     quantity,
		TradeAmount:  tradeAmount,
		UnitPrice:    unitPrice,
		AverageCost:  averageCost,
		ProfitLoss:   profitLoss,
		OccurrenceNo: 0, // assignOccurrenceNo で上書きされる
		Source:       "sbi",
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// splitBrand 「銘柄名 1234」形式を銘柄名と4桁コードに分離する。
// 銘柄名内にスペースが入る場合があるため、末尾の4桁を基点に分離する。
func splitBrand(v string) (name, code string, err error) {
	s := strings.TrimSpace(v)
	m := brandRe.FindStringSubmatch(s)
	if m == nil {
		return "", "", errors.Errorf("brand format unexpected: %q", s)
	}
	return strings.TrimSpace(m[1]), m[2], nil
}

func parseUintComma(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	v, err := strconv.ParseUint(s, 10, 32)
	return uint32(v), err
}

func parseIntComma(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseInt(s, 10, 64)
}

// parseSignedIntComma "+1,460" / "-740" / "0" を int64 に変換する。
func parseSignedIntComma(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimPrefix(s, "+")
	return strconv.ParseInt(s, 10, 64)
}

func parseDecimalComma(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	return decimal.NewFromString(s)
}
