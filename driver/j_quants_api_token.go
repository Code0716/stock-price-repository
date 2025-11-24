//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/config"
)

// idTokenは24時間有効
func (c *StockAPIClient) GetOrSetJQuantsAPIIDTokenToRedis(ctx context.Context) (string, error) {
	idToken, err := c.redisClient.Get(ctx, jQuantsAPIIDTokenRedisKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", errors.Wrap(err, "redisClient.Get error")
	}

	if errors.Is(err, redis.Nil) {
		// refreshTokenの取得
		refreshToken, err := c.getOrSetJQuantsAPIRefreshTokenToRedis(ctx)
		if err != nil {
			return "", errors.Wrap(err, "GetOrSetJQuantsAPIRefreshTokenToRedis error")
		}

		idToken, err = c.setIDToken(ctx, refreshToken)
		if err != nil {
			return "", errors.Wrap(err, "setIDToken error")
		}
	}

	return idToken, nil
}

// refreshTokenは一週間有効
func (c *StockAPIClient) getOrSetJQuantsAPIRefreshTokenToRedis(ctx context.Context) (string, error) {
	refreshToken, err := c.redisClient.Get(ctx, jQuantsAPIRefreshTokenRedisKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", errors.Wrap(err, "redisClient.Get error")
	}

	if errors.Is(err, redis.Nil) {
		refreshToken, err = c.setRefreshToken(ctx)
		if err != nil {
			return "", errors.Wrap(err, "setRefreshToken error")
		}
	}
	return refreshToken, nil
}

// idTokenをRedisにセット
func (c *StockAPIClient) setJQuantsAPIIDTokenToRedis(ctx context.Context, idToken string) error {
	err := c.redisClient.SetEx(ctx, jQuantsAPIIDTokenRedisKey, idToken, jQuantsAPIIDTokenRedisDuration).Err()
	if err != nil {
		return errors.Wrap(err, "setJQuantsAPIIDTokenToRedis error")
	}
	return nil
}

// refreshTokenをRedisにセット
func (c *StockAPIClient) setJQuantsAPIRefreshTokenToRedis(ctx context.Context, refreshToken string) error {
	err := c.redisClient.SetEx(ctx, jQuantsAPIRefreshTokenRedisKey, refreshToken, jQuantsAPIRefreshTokenRedisDuration).Err()
	if err != nil {
		return errors.Wrap(err, "setJQuantsAPIRefreshTokenToRedis error")
	}
	return nil
}

func (c *StockAPIClient) getNewIDToken(ctx context.Context) (string, error) {
	refreshToken, err := c.setRefreshToken(ctx)
	if err != nil {
		return "", errors.Wrap(err, "setRefreshToken error")
	}

	idToken, err := c.setIDToken(ctx, refreshToken)
	if err != nil {
		return "", errors.Wrap(err, "setIDToken error")
	}
	return idToken, nil
}

func (c *StockAPIClient) setRefreshToken(ctx context.Context) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/token/auth_user", config.GetJQuants().JQuantsBaseURLV1))
	if err != nil {
		return "", errors.Wrap(err, "JQuantsAPIClient.GetRefreshToken error")
	}

	b := jQuantsAPIClientRefreshTokenRequest{
		Mailaddress: config.GetJQuants().JQuantsMailaddress,
		Password:    config.GetJQuants().JQuantsPassword,
	}

	body, err := json.Marshal(b)
	if err != nil {
		return "", errors.Wrap(err, "json.Marshal error")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return "", errors.Wrap(err, "http.NewRequestWithContext error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf(`j-quants.api request to request to: %s`, u.String()))
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "j-quants.api  io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf(`j-quants.api  status error status: %d, url: %s`, res.StatusCode, u.String())
	}

	var response jQuantsAPIClientRefreshTokenResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return "", errors.Wrap(err, fmt.Sprintf(`j-quants.api request to: %s`, u.String()))
	}

	err = c.setJQuantsAPIRefreshTokenToRedis(ctx, response.RefreshToken)
	if err != nil {
		return "", errors.Wrap(err, "setRefreshToken error")
	}
	return response.RefreshToken, nil
}

func (c *StockAPIClient) setIDToken(ctx context.Context, refreshToken string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/token/auth_refresh?refreshtoken=%s", config.GetJQuants().JQuantsBaseURLV1, refreshToken))
	if err != nil {
		return "", errors.Wrap(err, "JQuantsAPIClient.getIDToken error")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", errors.Wrap(err, "http.NewRequestWithContext error")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	res, err := c.request.GetHTTPClient().Do(req)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf(`j-quants.api request to request to: %s/token/auth_refresh?refreshtoken=`, u.Host))
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "j-quants.api  io.ReadAll error")
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf(`j-quants.api  status error status: %d, url: %s`, res.StatusCode, u.String())
	}

	var response jQuantsAPIClientIDTokenResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		log.Printf("JSON parse error: %v", err)
		return "", errors.Wrap(err, fmt.Sprintf(`j-quants.api  request to: %s`, u.String()))
	}

	err = c.setJQuantsAPIIDTokenToRedis(ctx, response.IDToken)
	if err != nil {
		return "", errors.Wrap(err, "setJQuantsAPIIDTokenToRedis error")
	}
	return response.IDToken, nil
}
