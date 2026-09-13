package domain_service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"

	"github.com/Code0716/stock-price-repository/models"
)

// DailyPickSlackMaxRunes Slack 1メッセージあたりの文字数上限（4000文字に対する安全マージン込み）。
const DailyPickSlackMaxRunes = 3500

// FormatDailyStockPickMessages 推奨銘柄を Slack の親タイトル＋スレッド本文群に整形する。
// bodies は各要素が maxRunes 以下になるよう銘柄ブロック単位で貪欲に詰める（銘柄が途中で切れない）。
// title は SendMessageByStrings 側で *%s* に包まれるため、自分で強調記号を付けない。
func FormatDailyStockPickMessages(picks []*models.DailyStockPick, maxRunes int) (title string, bodies []string) {
	if len(picks) == 0 {
		return "", nil
	}

	title = formatDailyStockPickTitle(picks)

	blocks := make([]string, len(picks))
	for i, p := range picks {
		blocks[i] = formatDailyStockPickBlock(i+1, p)
	}

	legend := dailyStockPickLegend()

	chunks := chunkDailyStockPickBlocks(blocks, maxRunes)
	bodies = make([]string, 0, len(chunks))
	for i, c := range chunks {
		heading := fmt.Sprintf("*%d〜%d位*\n\n", c.startRank, c.endRank)
		bodies = append(bodies, heading+strings.Join(c.blocks, "\n\n"))
		if i == len(chunks)-1 {
			last := bodies[len(bodies)-1]
			if utf8.RuneCountInString(last)+utf8.RuneCountInString(legend) <= maxRunes {
				bodies[len(bodies)-1] = last + "\n\n" + legend
			} else {
				bodies = append(bodies, legend)
			}
		}
	}
	return title, bodies
}

func formatDailyStockPickTitle(picks []*models.DailyStockPick) string {
	best := picks[0]
	sum := decimal.Zero
	for _, p := range picks {
		sum = sum.Add(p.Score)
	}
	avg := sum.Div(decimalFromInt(len(picks))).Round(1)
	return fmt.Sprintf(
		"翌営業日の買い候補 %s引け後 / %d銘柄 / 平均スコア %s / 最高 %s %s",
		picks[0].PickDate.Format("2006-01-02"),
		len(picks),
		avg.String(),
		best.TickerSymbol,
		best.Score.String(),
	)
}

type dailyStockPickChunk struct {
	startRank, endRank int
	blocks             []string
}

// chunkDailyStockPickBlocks 銘柄ブロックを見出し込みで maxRunes 以下になるよう貪欲に詰める。
// 銘柄ブロック自体は絶対に分割しない。
func chunkDailyStockPickBlocks(blocks []string, maxRunes int) []dailyStockPickChunk {
	var chunks []dailyStockPickChunk
	start := 0
	for start < len(blocks) {
		headingOverhead := utf8.RuneCountInString("*000〜000位*\n\n")
		size := headingOverhead
		end := start
		for end < len(blocks) {
			blockLen := utf8.RuneCountInString(blocks[end])
			sep := 0
			if end > start {
				sep = utf8.RuneCountInString("\n\n")
			}
			if end > start && size+sep+blockLen > maxRunes {
				break
			}
			size += sep + blockLen
			end++
		}
		if end == start {
			end = start + 1 // 1ブロックがmaxRunesを超えても必ず1つは詰める
		}
		chunks = append(chunks, dailyStockPickChunk{
			startRank: start + 1,
			endRank:   end,
			blocks:    blocks[start:end],
		})
		start = end
	}
	return chunks
}

func formatDailyStockPickBlock(rank int, p *models.DailyStockPick) string {
	strategyLabels := make([]string, 0, len(p.Strategies))
	for _, s := range p.Strategies {
		if label, ok := StrategyLabels[s]; ok {
			strategyLabels = append(strategyLabels, label)
		} else {
			strategyLabels = append(strategyLabels, s)
		}
	}

	sector := p.Sector33CodeName
	sectorPart := ""
	if sector != "" {
		sectorPart = fmt.Sprintf(" 〔%s〕", escapeSlackText(sector))
	}

	return fmt.Sprintf(
		"*%d.* `%s` %s%s\n"+
			"    %s円 / score *%s* / %d戦略: %s\n"+
			"    出来高 %s倍 / ADX %s / ATR %s%% / RSI %s / 平均代金 %s億円",
		rank,
		p.TickerSymbol,
		escapeSlackText(p.Name),
		sectorPart,
		formatDecimalComma(p.BaseClosePrice),
		p.Score.StringFixed(1),
		len(p.Strategies),
		strings.Join(strategyLabels, "・"),
		p.VolumeRatio.StringFixed(1),
		p.ADX.StringFixed(1),
		p.ATRRatio.Mul(decimalFromInt(100)).StringFixed(1),
		p.RSI.StringFixed(1),
		p.AvgTradingValue.Div(decimalFromInt(100000000)).StringFixed(1),
	)
}

func dailyStockPickLegend() string {
	return "─────\n" +
		"score = 点灯戦略数30 + 出来高急増20 + ADX20 + ATR適正10 + 流動性10 + RSI10（" + DailyPickScoreVersion + "）\n" +
		"条件: 主要市場 / 300円以上 / 20日平均売買代金1億円以上 / +DI>-DI / 同一業種は最大4銘柄\n" +
		"5営業日後に自動で答え合わせします。投資判断は自己責任で。"
}

// FormatDailyStockPickResultMessage 答え合わせサマリを1メッセージに整形する。
func FormatDailyStockPickResultMessage(s *models.DailyStockPickSummary) string {
	if s == nil || s.Total == 0 {
		return ""
	}
	lines := []string{
		fmt.Sprintf("買い候補の答え合わせ %s推奨分 / %d銘柄", s.PickDate.Format("2006-01-02"), s.Total),
		"",
		fmt.Sprintf(
			"勝敗: %d勝 %d敗 %d分 %d除外 (勝率 %s%%)",
			s.Win, s.Lose, s.Draw, s.Void, s.WinRate.Mul(decimalFromInt(100)).StringFixed(1),
		),
		fmt.Sprintf(
			"平均リターン: 1日 %s%% / 5日 %s%%",
			formatSignedPercent(s.AvgReturn1D), formatSignedPercent(s.AvgReturn5D),
		),
	}
	if s.Best != nil && s.Best.Return5D != nil {
		lines = append(lines, fmt.Sprintf(
			"ベスト: `%s` %s %s%%",
			s.Best.TickerSymbol, escapeSlackText(s.Best.Name), formatSignedPercent(*s.Best.Return5D),
		))
	}
	if s.Worst != nil && s.Worst.Return5D != nil {
		lines = append(lines, fmt.Sprintf(
			"ワースト: `%s` %s %s%%",
			s.Worst.TickerSymbol, escapeSlackText(s.Worst.Name), formatSignedPercent(*s.Worst.Return5D),
		))
	}
	return strings.Join(lines, "\n")
}

// escapeSlackText Slack の特殊文字（& < >）をエスケープする。銘柄名対策。
func escapeSlackText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func decimalFromInt(n int) decimal.Decimal {
	return decimal.NewFromInt(int64(n))
}

// formatDecimalComma 小数を四捨五入し、3桁区切りカンマを付けた文字列にする（株価表示用）。
func formatDecimalComma(d decimal.Decimal) string {
	s := d.Round(0).StringFixed(0)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	first := n % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(s[:first])
	for i := first; i < n; i += 3 {
		b.WriteString(",")
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// formatSignedPercent リターン比率（例: 0.0287）を符号付きパーセント文字列（例: "+2.9"）にする。
func formatSignedPercent(d decimal.Decimal) string {
	pct := d.Mul(decimalFromInt(100)).Round(1)
	if pct.IsNegative() {
		return pct.StringFixed(1)
	}
	return "+" + pct.StringFixed(1)
}
