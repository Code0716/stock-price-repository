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

// ParseSBIDaytradeCSV SBI証券の国内株式取引履歴CSVをパースしてドメインモデルに変換する。
// CSVはShift-JIS(CP932)で出力される。UTF-8 BOMがあればUTF-8として扱う。
// データ行は14行目以降 (1〜13行目はメタ情報/ヘッダー)。
// CSVは約定日の降順 (上が新しい/下が古い) だが、挿入順序には依存しない。
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
		if len(record) < dataColumns {
			continue
		}
		// データ行は先頭列が "YYYY/M/D" 形式の日付。それ以外はメタ/ヘッダー行としてスキップ。
		if _, dateErr := time.ParseInLocation(executedOnLayout, strings.TrimSpace(record[0]), time.Local); dateErr != nil {
			continue
		}
		ex, err := buildExecution(record, now)
		if err != nil {
			return nil, wrapParseError(errors.Wrapf(err, "csv data record %d", recNo))
		}
		executions = append(executions, ex)
	}
	return executions, nil
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

func buildExecution(record []string, now time.Time) (*models.DaytradeExecution, error) {
	executedOn, err := time.ParseInLocation("2006/1/2", strings.TrimSpace(record[0]), time.Local)
	if err != nil {
		return nil, errors.Wrap(err, "executedOn")
	}

	tradeKind := strings.TrimSpace(record[1])

	brandName, tickerSymbol, err := splitBrand(record[2])
	if err != nil {
		return nil, err
	}

	marginKind := strings.TrimSpace(record[3])

	quantity, err := parseUintComma(record[4])
	if err != nil {
		return nil, errors.Wrap(err, "quantity")
	}

	tradeAmount, err := parseIntComma(record[5])
	if err != nil {
		return nil, errors.Wrap(err, "tradeAmount")
	}

	unitPrice, err := parseDecimalComma(record[6])
	if err != nil {
		return nil, errors.Wrap(err, "unitPrice")
	}

	averageCost, err := parseDecimalComma(record[7])
	if err != nil {
		return nil, errors.Wrap(err, "averageCost")
	}

	profitLoss, err := parseSignedIntComma(record[8])
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
