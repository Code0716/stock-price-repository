package driver

import (
	"context"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
)

type EconomicCalendarClient struct {
	request HTTPRequest
}

func NewEconomicCalendarClient(request HTTPRequest) gateway.EconomicCalendarClient {
	return &EconomicCalendarClient{request: request}
}

func (c *EconomicCalendarClient) FetchBOJSchedulePage(ctx context.Context) ([]byte, error) {
	return c.fetch(ctx, config.GetEconomicCalendar().BOJScheduleURL)
}

func (c *EconomicCalendarClient) FetchFOMCPage(ctx context.Context) ([]byte, error) {
	return c.fetch(ctx, config.GetEconomicCalendar().FOMCCalendarURL)
}

func (c *EconomicCalendarClient) FetchBLSICS(ctx context.Context) ([]byte, error) {
	return c.fetch(ctx, config.GetEconomicCalendar().BLSICSURL)
}

// fetch URL を GET して body を返す。BLS は User-Agent が無いと拒否されるため必ず付ける。
func (c *EconomicCalendarClient) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "EconomicCalendarClient.fetch request error")
	}
	req.Header.Set("User-Agent", config.GetEconomicCalendar().HTTPUserAgent)

	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "EconomicCalendarClient.fetch do error: "+url)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "EconomicCalendarClient.fetch io.ReadAll error")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("EconomicCalendarClient.fetch status error status: %d, url: %s", res.StatusCode, url)
	}
	return body, nil
}
