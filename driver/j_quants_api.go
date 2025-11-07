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

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/util"
	holidayJP "github.com/holiday-jp/holiday_jp-go"
	"github.com/pkg/errors"
)

const (
	jQuantsAPIRefreshTokenRedisKey      string        = "j_quants_api_refresh_token_redis_key"
	jQuantsAPIIDTokenRedisKey           string        = "j_quants_api_id_token_redis_key"
	jQuantsAPIRefreshTokenRedisDuration time.Duration = 7 * 24 * time.Hour
	jQuantsAPIIDTokenRedisDuration      time.Duration = 24 * time.Hour
)

func (jc *StockAPIClient) GetStockBrands(ctx context.Context) ([]*gateway.StockBrand, error) {
	idToken, err := jc.GetOrSetJQuantsAPIIDTokenToRedis(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetOrSetJQuantsAPIIDTokenToRedis error")
	}
	u, err := url.Parse(fmt.Sprintf("%s/listed/info", config.JQuants().JQuantsBaseURLV1))
	if err != nil {
		return nil, errors.Wrap(err, u.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api request error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", "SttApp/1.0 Go-http-client/1.1 (linux)")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", idToken))

	res, err := jc.request.GetHttpClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	if res.StatusCode == http.StatusUnauthorized {
		// IDToken再取得
		_, err := jc.getNewIDToken(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getNewIDToken error")
		}
		// 再度リクエスト。だめだったらエラーを返す。
		result, err := jc.GetStockBrands(ctx)
		return result, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String()))
	}

	var response jQuantsStockBrandsResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request error to: %s`, u.String()))
	}
	responseInfo := jc.jQuantsStockBrandsResponseToResponseInfo(response)

	return responseInfo, nil
}

func (jc *StockAPIClient) GetAnnounceFinsSchedule(ctx context.Context) ([]*gateway.AnnounceFinScheduleResponseInfo, error) {
	idToken, err := jc.GetOrSetJQuantsAPIIDTokenToRedis(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetOrSetJQuantsAPIIDTokenToRedis error")
	}
	u, err := url.Parse(fmt.Sprintf("%s/fins/announcement", config.JQuants().JQuantsBaseURLV1))
	if err != nil {
		return nil, errors.Wrap(err, u.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api request error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", idToken))

	res, err := jc.request.GetHttpClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	if res.StatusCode == http.StatusUnauthorized {
		// IDToken再取得
		_, err := jc.getNewIDToken(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getNewIDToken error")
		}
		// 再度リクエスト。だめだったらエラーを返す。
		result, err := jc.GetAnnounceFinsSchedule(ctx)
		return result, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String()))
	}

	var response jQuantsAnnounceFinsScheduleResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	responseInfo := jc.jQuantsAnnounceFinsScheduleResponseToResponseInfo(response)

	return responseInfo, nil
}

// 場中の価格が取れるわけではない
func (c *StockAPIClient) getDailyPricesBySymbolAndRangeJQ(ctx context.Context, symbol string, dateFrom, dateTo time.Time) ([]*gateway.StockPrice, error) {
	idToken, err := c.GetOrSetJQuantsAPIIDTokenToRedis(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetCurrentStockPriceBySymbol error")
	}

	// 念の為、土日祝日だったら前の日にする。
	lastWeekday := c.getLastWeekday(dateTo)
	u, err := url.Parse(
		fmt.Sprintf(
			"%s/prices/daily_quotes?code=%s&from=%s&to=%s",
			config.JQuants().JQuantsBaseURLV1,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", idToken))

	res, err := c.request.GetHttpClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	if res.StatusCode == http.StatusUnauthorized {
		// IDToken再取得
		_, err := c.getNewIDToken(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getNewIDToken error")
		}
		// 再度リクエスト。だめだったらエラーを返す。
		result, err := c.getDailyPricesBySymbolAndRangeJQ(ctx, symbol, dateFrom, dateTo)
		return result, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String()))
	}

	var response jQuantsDailyQuotesResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("json parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	responseInfo := c.jQuantsDailyQuotesResponseToResponseInfo(response)

	return responseInfo, nil
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

func (jc *StockAPIClient) getFinancialStatementsJQ(ctx context.Context, symbol string, date *time.Time) ([]*gateway.FinancialStatementsResponseInfo, error) {
	idToken, err := jc.GetOrSetJQuantsAPIIDTokenToRedis(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetOrSetJQuantsAPIIDTokenToRedis error")
	}

	u, err := url.Parse(fmt.Sprintf("%s/fins/statements?code=%s", config.JQuants().JQuantsBaseURLV1, symbol))
	if err != nil {
		return nil, errors.Wrap(err, u.String())
	}

	if date != nil {
		u, err = url.Parse(
			fmt.Sprintf("%s/fins/statements?date=%s",
				config.JQuants().JQuantsBaseURLV1,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", idToken))

	res, err := jc.request.GetHttpClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	if res.StatusCode == http.StatusUnauthorized {
		// IDToken再取得
		_, err := jc.getNewIDToken(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getNewIDToken error")
		}
		// 再度リクエスト。だめだったらエラーを返す。
		result, err := jc.getFinancialStatementsJQ(ctx, symbol, date)
		return result, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String()))
	}

	var response jQuantsFinancialStatementsResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request error to: %s`, u.String()))
	}
	responseInfo := jc.jQuantsFinancialStatementsToGatewayModels(response)

	return responseInfo, nil
}

func (c *StockAPIClient) getTradingCalendarsInfo(ctx context.Context, filter gateway.TradingCalendarsInfoFilter) ([]*gateway.TradingCalendarsInfo, error) {
	idToken, err := c.GetOrSetJQuantsAPIIDTokenToRedis(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "GetOrSetJQuantsAPIIDTokenToRedis error")
	}

	u, err := url.Parse(fmt.Sprintf(
		"%s/markets/trading_calendar?from=%s&to=%s",
		config.JQuants().JQuantsBaseURLV1,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", idToken))

	res, err := c.request.GetHttpClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}
	if res.StatusCode == http.StatusUnauthorized {
		// IDToken再取得
		_, err := c.getNewIDToken(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getNewIDToken error")
		}
		// 再度リクエスト。だめだったらエラーを返す。
		result, err := c.getTradingCalendarsInfo(ctx, filter)
		return result, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "j-quants.api io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf(`j-quants.api status error status: %d, url: %s`, res.StatusCode, u.String()))
	}

	var response TradingCalendarsResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.Wrap(err, fmt.Sprintf(`j-quants.api request error to: %s`, u.String()))
	}

	return c.jQuantsTradingCalendarsToGatewayModels(response), nil
}
