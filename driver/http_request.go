//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type HTTPRequest interface {
	Get(ctx context.Context, url string, values *url.Values) ([]byte, error)
	Post(ctx context.Context, url string, values url.Values) ([]byte, error)
	GetHTTPClient() *http.Client
}

type HTTPRequestImpl struct{}

func NewHTTPRequest() HTTPRequest {
	return &HTTPRequestImpl{}
}

var sharedHTTPClientInstance = newHTTPClient()

func newHTTPClient() *http.Client {
	return &http.Client{}
}

func (r HTTPRequestImpl) GetHTTPClient() *http.Client {
	return sharedHTTPClientInstance
}

func (r HTTPRequestImpl) Get(ctx context.Context, url string, values *url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	if values != nil {
		req.URL.RawQuery = values.Encode()
	}

	client := r.GetHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer resp.Body.Close()

	v, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return v, nil
}

func (r HTTPRequestImpl) Post(ctx context.Context, url string, values url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	req.Header.Set("Content-Type", "application/json")

	client := r.GetHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer resp.Body.Close()

	v, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return v, nil
}
