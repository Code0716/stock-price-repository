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
	"time"

	holidayJP "github.com/holiday-jp/holiday_jp-go"
	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/util"
)

func (c *StockAPIClient) GetStockBrands(ctx context.Context) ([]*gateway.StockBrand, error) {

	u, err := url.Parse(fmt.Sprintf("%s/equities/master", config.GetJQuants().JQuantsBaseURLV2))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api request error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", "SttApp/1.0 Go-http-client/1.1 (linux)")
	req.Header.Set("x-api-key", config.GetJQuants().JQuantsBaseURLV2APIKey)
	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.Wrap(err, "http StatusUnauthorized error")
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String())
	}

	var response jQuantsStockBrandsResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request error to: %s`, u.String()))
	}
	responseInfo := c.jQuantsStockBrandsResponseToResponseInfo(response)

	return responseInfo, nil
}

func (c *StockAPIClient) GetAnnounceFinSchedule(ctx context.Context) ([]*gateway.AnnounceFinScheduleResponseInfo, error) {
	u, err := url.Parse(fmt.Sprintf("%s/equities/earnings-calendar", config.GetJQuants().JQuantsBaseURLV2))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api request error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("x-api-key", config.GetJQuants().JQuantsBaseURLV2APIKey)

	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.Wrap(err, "http StatusUnauthorized error")
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String())
	}

	var response jQuantsAnnounceFinsScheduleResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	responseInfo := c.jQuantsAnnounceFinsScheduleResponseToResponseInfo(response)

	return responseInfo, nil
}

// getDailyPricesBySymbolAndRangeJQ - 指定した証券コードの日足を指定した期間分取得する
// 場中の価格が取れるわけではない
func (c *StockAPIClient) getDailyPricesBySymbolAndRangeJQ(ctx context.Context, symbol string, dateFrom, dateTo time.Time) ([]*gateway.StockPrice, error) {
	// 念の為、土日祝日だったら前の日にする。
	lastWeekday := c.getLastWeekday(dateTo)
	u, err := url.Parse(
		fmt.Sprintf(
			"%s/equities/bars/daily?code=%s&from=%s&to=%s",
			config.GetJQuants().JQuantsBaseURLV2,
			symbol,
			util.DatetimeToDateStr(dateFrom),
			util.DatetimeToDateStr(lastWeekday),
		))
	if err != nil {
		return nil, errors.Wrap(err, u.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api request error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("x-api-key", config.GetJQuants().JQuantsBaseURLV2APIKey)

	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.Wrap(err, "http StatusUnauthorized error")
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String())
	}

	var response jQuantsDailyQuotesResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("json parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	responseInfo := c.jQuantsDailyQuotesResponseToResponseInfo(response)

	return responseInfo, nil
}

// getAllBrandDailyPricesByDate - すべての銘柄の指定した日の日足を取得する（ページネーション対応）
func (c *StockAPIClient) getAllBrandDailyPricesByDate(ctx context.Context, date time.Time) ([]*gateway.StockPrice, error) {
	var allPrices []*gateway.StockPrice
	paginationKey := ""

	for {
		rawURL := fmt.Sprintf("%s/equities/bars/daily?date=%s", config.GetJQuants().JQuantsBaseURLV2, util.DatetimeToDateStr(date))
		if paginationKey != "" {
			rawURL += "&pagination_key=" + url.QueryEscape(paginationKey)
		}

		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, errors.Wrap(err, u.String())
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, errors.Wrap(err, "j-quants.api request error")
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json;charset=UTF-8")
		req.Header.Set("x-api-key", config.GetJQuants().JQuantsBaseURLV2APIKey)

		res, err := c.request.GetHTTPClient().Do(req)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
		}

		if res.StatusCode == http.StatusUnauthorized {
			res.Body.Close()
			return nil, errors.New("http StatusUnauthorized error")
		}

		// 210: 祝日など取引がない日
		if res.StatusCode == 210 {
			res.Body.Close()
			return nil, nil
		}

		resBody, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
		}

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String())
		}

		var response jQuantsDailyQuotesResponse
		if err := json.Unmarshal(resBody, &response); err != nil {
			log.Printf("json parse error: %v", err)
			return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
		}

		allPrices = append(allPrices, c.jQuantsDailyQuotesResponseToResponseInfo(response)...)

		if response.PaginationKey == "" {
			break
		}
		paginationKey = response.PaginationKey
	}

	return allPrices, nil
}

func (c *StockAPIClient) getLastWeekday(t time.Time) time.Time {
	for {
		// 祝日でなく、土日でなかったら日を返す。
		if !holidayJP.IsHoliday(t) && t.Weekday() != time.Saturday && t.Weekday() != time.Sunday {
			return t
		}
		// 土日か祝日なら前の日に戻る
		t = t.AddDate(0, 0, -1)
	}
}

func (c *StockAPIClient) getFinancialStatementsJQ(ctx context.Context, symbol string, date *time.Time) ([]*gateway.FinancialStatementsResponseInfo, error) {
	u, err := url.Parse(fmt.Sprintf("%s/fins/summary?code=%s", config.GetJQuants().JQuantsBaseURLV2, symbol))
	if err != nil {
		return nil, errors.Wrap(err, u.String())
	}

	if date != nil {
		u, err = url.Parse(
			fmt.Sprintf("%s/fins/summary?date=%s",
				config.GetJQuants().JQuantsBaseURLV2,
				util.DatetimeToDateStr(
					util.FromPtrGenerics(date),
				),
			),
		)
		if err != nil {
			return nil, errors.Wrap(err, u.String())
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api request error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("x-api-key", config.GetJQuants().JQuantsBaseURLV2APIKey)

	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}

	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.Wrap(err, "http StatusUnauthorized error")
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String())
	}

	var response jQuantsFinancialStatementsResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request error to: %s`, u.String()))
	}
	responseInfo := c.jQuantsFinancialStatementsToGatewayModels(response)

	return responseInfo, nil
}

func (c *StockAPIClient) getTradingCalendarsInfo(ctx context.Context, filter gateway.TradingCalendarsInfoFilter) ([]*gateway.TradingCalendarsInfo, error) {

	u, err := url.Parse(fmt.Sprintf(
		"%s/markets/calendar?from=%s&to=%s",
		config.GetJQuants().JQuantsBaseURLV2,
		util.DatetimeToDateStr(filter.From),
		util.DatetimeToDateStr(filter.To),
	),
	)
	if err != nil {
		return nil, errors.Wrap(err, u.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api request error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("x-api-key", config.GetJQuants().JQuantsBaseURLV2APIKey)

	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	if res.StatusCode == http.StatusUnauthorized {

		return nil, errors.Wrap(err, "http StatusUnauthorized error")
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String())
	}

	var response TradingCalendarsResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request error to: %s`, u.String()))
	}

	return c.jQuantsTradingCalendarsToGatewayModels(response), nil
}
