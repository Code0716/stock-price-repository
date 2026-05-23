package daytrade

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testNow = time.Date(2026, 5, 22, 0, 0, 0, 0, time.Local)

func TestParseSBIDaytradeCSV_ShiftJIS_HappyPath(t *testing.T) {
	f, err := os.Open("testdata/sbi_sample_sjis.csv")
	require.NoError(t, err)
	defer f.Close()

	got, err := ParseSBIDaytradeCSV(f, testNow)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// 1行目: ソフトバンクグループ 9984
	e := got[0]
	assert.Equal(t, "9984", e.TickerSymbol)
	assert.Equal(t, "ソフトバンクグループ", e.BrandName)
	assert.Equal(t, "売建", e.TradeKind)
	assert.Equal(t, "返済売", e.MarginKind)
	assert.Equal(t, uint32(200), e.Quantity)
	assert.Equal(t, int64(1188860), e.TradeAmount)
	assert.Equal(t, int64(1460), e.ProfitLoss)
	assert.True(t, decimal.NewFromFloat(5944.3).Equal(e.UnitPrice), "unitPrice mismatch: %s", e.UnitPrice)
	assert.True(t, decimal.NewFromFloat(5937).Equal(e.AverageCost), "averageCost mismatch: %s", e.AverageCost)
	assert.Equal(t, "sbi", e.Source)
	assert.Equal(t, time.Date(2026, 5, 21, 0, 0, 0, 0, time.Local), e.ExecutedOn)

	// 2行目: マイナス損益
	assert.Equal(t, int64(-800), got[1].ProfitLoss)

	// 3行目: 別日付
	assert.Equal(t, time.Date(2026, 5, 19, 0, 0, 0, 0, time.Local), got[2].ExecutedOn)
	assert.Equal(t, int64(2040), got[2].ProfitLoss)
}

func TestParseSBIDaytradeCSV_UTF8BOM_Fallback(t *testing.T) {
	f, err := os.Open("testdata/sbi_sample_utf8_bom.csv")
	require.NoError(t, err)
	defer f.Close()

	got, err := ParseSBIDaytradeCSV(f, testNow)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, "ソフトバンクグループ", got[0].BrandName)
}

func TestParseSBIDaytradeCSV_SignedProfitLoss(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{"plus sign", "+1,460", 1460},
		{"minus sign", "-740", -740},
		{"zero", "0", 0},
		{"large plus", "+10,600", 10600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSignedIntComma(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSBIDaytradeCSV_DecimalUnit(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"5,944.3", "5944.3"},
		{"5,937", "5937"},
		{"4,395", "4395"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDecimalComma(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestParseSBIDaytradeCSV_BrandSplit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantCode string
		wantErr  bool
	}{
		{
			name:     "standard",
			input:    "ソフトバンクグループ 9984",
			wantName: "ソフトバンクグループ",
			wantCode: "9984",
		},
		{
			name:     "brand with space in name",
			input:    "三菱 UFJ フィナンシャル グループ 8306",
			wantName: "三菱 UFJ フィナンシャル グループ",
			wantCode: "8306",
		},
		{
			name:     "new format alphanumeric code (e.g. Kioxia 285A)",
			input:    "キオクシアホールディングス 285A",
			wantName: "キオクシアホールディングス",
			wantCode: "285A",
		},
		{
			name:     "new format alphanumeric code 130A",
			input:    "テスト会社 130A",
			wantName: "テスト会社",
			wantCode: "130A",
		},
		{
			name:    "lowercase code is rejected",
			input:   "テスト会社 285a",
			wantErr: true,
		},
		{
			name:    "no code",
			input:   "銘柄名のみ",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, code, err := splitBrand(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantCode, code)
		})
	}
}

func TestParseSBIDaytradeCSV_SkipsMetaAndEmpty(t *testing.T) {
	// 13行のメタ + 空行 + データ1行
	content := "meta1\nmeta2\nmeta3\nmeta4\nmeta5\nmeta6\nmeta7\nmeta8\nmeta9\nmeta10\nmeta11\nmeta12\nheader\n\n" +
		`"2026/5/21","売建","ソフトバンクグループ 9984","返済売","100","580,000","5,800","5,790","+1,000"` + "\n"

	got, err := ParseSBIDaytradeCSV(bytes.NewReader([]byte(content)), testNow)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, int64(1000), got[0].ProfitLoss)
}

func TestParseSBIDaytradeCSV_AbortsOnUnparseable(t *testing.T) {
	content := "m\nm\nm\nm\nm\nm\nm\nm\nm\nm\nm\nm\nh\n" +
		`"2026/5/21","売建","ソフトバンクグループ 9984","返済売","100","580,000","INVALID","5,790","+1,000"` + "\n"

	_, err := ParseSBIDaytradeCSV(bytes.NewReader([]byte(content)), testNow)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrParse)
}

func TestParseSBIDaytradeCSV_RejectsUTF16BOM(t *testing.T) {
	// UTF-16 LE BOM
	data := []byte{0xFF, 0xFE, 0x00, 0x41}
	_, err := ParseSBIDaytradeCSV(bytes.NewReader(data), testNow)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrParse)
}

func TestParseSBIDaytradeCSV_LimitsSize(t *testing.T) {
	// 10MB超のデータ
	data := bytes.Repeat([]byte("a"), maxCSVBytes+2)
	_, err := ParseSBIDaytradeCSV(bytes.NewReader(data), testNow)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrParse)
}

func TestParseSBIDaytradeCSV_NewFormat_ShiftJIS(t *testing.T) {
	f, err := os.Open("testdata/sbi_sample_new_sjis.csv")
	require.NoError(t, err)
	defer f.Close()

	got, err := ParseSBIDaytradeCSV(f, testNow)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// 新フォーマット: tradeKind="" / marginKind=取引列の値
	e := got[0]
	assert.Equal(t, "9984", e.TickerSymbol)
	assert.Equal(t, "ソフトバンクグループ", e.BrandName)
	assert.Equal(t, "", e.TradeKind, "新フォーマットはtradeKindが空")
	assert.Equal(t, "返済売", e.MarginKind)
	assert.Equal(t, uint32(200), e.Quantity)
	assert.Equal(t, int64(1188860), e.TradeAmount)
	assert.Equal(t, int64(1460), e.ProfitLoss)
	assert.True(t, decimal.NewFromFloat(5944.3).Equal(e.UnitPrice), "unitPrice mismatch: %s", e.UnitPrice)
	assert.True(t, decimal.NewFromFloat(5937).Equal(e.AverageCost), "averageCost mismatch: %s", e.AverageCost)
	assert.Equal(t, "sbi", e.Source)
	assert.Equal(t, time.Date(2026, 5, 21, 0, 0, 0, 0, time.Local), e.ExecutedOn)

	// 2行目: マイナス損益
	assert.Equal(t, int64(-800), got[1].ProfitLoss)
	assert.Equal(t, "", got[1].TradeKind)

	// 3行目: 返済買
	assert.Equal(t, "返済買", got[2].MarginKind)
}

func TestParseSBIDaytradeCSV_OccurrenceNo_UniqueRows(t *testing.T) {
	// 重複のない場合は全行 occurrence_no = 0
	f, err := os.Open("testdata/sbi_sample_sjis.csv")
	require.NoError(t, err)
	defer f.Close()

	got, err := ParseSBIDaytradeCSV(f, testNow)
	require.NoError(t, err)
	for i, e := range got {
		assert.Equal(t, uint32(0), e.OccurrenceNo, "row %d should have occurrence_no=0", i)
	}
}

func TestParseSBIDaytradeCSV_OccurrenceNo_Duplicates(t *testing.T) {
	// 同一自然キーが3行ある場合 → 0, 1, 2 と採番される
	header := `"約定日","取引","銘柄","信用","数量","約定代金","単価","平均取得単価","売買損益(税引前・円)"` + "\n"
	row := `"2026/5/14","売建","ソフトバンクグループ 9984","返済売","100","596,940","5,969.4","5,960","+840"` + "\n"
	rowDiff := `"2026/5/14","売建","ソフトバンクグループ 9984","返済売","100","596,940","5,969.4","5,960","-100"` + "\n"
	content := header + row + row + row + rowDiff

	got, err := ParseSBIDaytradeCSV(bytes.NewReader([]byte(content)), testNow)
	require.NoError(t, err)
	require.Len(t, got, 4)

	// 最初の3行は同一キー → 0, 1, 2
	assert.Equal(t, uint32(0), got[0].OccurrenceNo)
	assert.Equal(t, uint32(1), got[1].OccurrenceNo)
	assert.Equal(t, uint32(2), got[2].OccurrenceNo)
	// 4行目は損益が違う別キー → 0
	assert.Equal(t, uint32(0), got[3].OccurrenceNo)
}
