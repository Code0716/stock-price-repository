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
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
)

type SlackAPIClient struct {
	request     HTTPRequest
	redisClient *redis.Client
}

func NewSlackAPIClient(
	request HTTPRequest,
	redisClient *redis.Client,
) gateway.SlackAPIClientRaw {
	return &SlackAPIClient{
		request,
		redisClient,
	}
}

type slackResponse struct {
	Ok      bool   `json:"ok"`
	Channel string `json:"channel"`
	Ts      string `json:"ts"`
	Message struct {
		User       string `json:"user"`
		Type       string `json:"type"`
		Ts         string `json:"ts"`
		BotID      string `json:"bot_id"`
		AppID      string `json:"app_id"`
		Text       string `json:"text"`
		Team       string `json:"team"`
		BotProfile struct {
			ID    string `json:"id"`
			AppID string `json:"app_id"`
			Name  string `json:"name"`
			Icons struct {
				Image36 string `json:"image_36"`
				Image48 string `json:"image_48"`
				Image72 string `json:"image_72"`
			} `json:"icons"`
			Deleted bool   `json:"deleted"`
			Updated int    `json:"updated"`
			TeamID  string `json:"team_id"`
		} `json:"bot_profile"`
		Blocks []struct {
			Type     string `json:"type"`
			BlockID  string `json:"block_id"`
			Elements []struct {
				Type     string `json:"type"`
				Elements []struct {
					Type  string `json:"type"`
					Range string `json:"range,omitempty"`
					Text  string `json:"text,omitempty"`
				} `json:"elements"`
			} `json:"elements"`
		} `json:"blocks"`
	} `json:"message"`
	Error string `json:"error,omitempty"`
}

func (c *SlackAPIClient) SendMessage(ctx context.Context, channelName gateway.SlackChannelName, message resource.SlackMessage) error {
	_, err := c.sendMessage(ctx, channelName, message.GetMessage(), nil)
	return err
}

func (c *SlackAPIClient) SendErrMessageNotification(ctx context.Context, err error) error {
	_, slackErr := c.sendMessage(ctx, gateway.SlackChannelNameDevNotification, err.Error(), nil)
	return slackErr
}

// tsがなければ、タイトルを送信して、そのスレにmassageを投稿する。tsがあれば、threadへの追記をおこなう。
func (c *SlackAPIClient) SendMessageByStrings(ctx context.Context, channelName gateway.SlackChannelName, title string, message, ts *string) (string, error) {
	threadTs := ts
	if threadTs == nil {
		postRes, err := c.sendMessage(ctx, channelName, fmt.Sprintf("<!channel> \n*%s*", title), nil)
		if err != nil {
			return "", errors.Wrap(err, "sendMessage error")
		}
		threadTs = &postRes.Ts
	}
	// タイトルだけなら、それでおわり。
	if message == nil {
		return *threadTs, nil
	}

	if _, err := c.sendMessage(ctx, channelName, *message, threadTs); err != nil {
		return "", errors.Wrap(err, "sendMessage error")
	}

	return *threadTs, nil
}

func (c *SlackAPIClient) sendMessage(ctx context.Context, channelName gateway.SlackChannelName, message string, replytimestamp *string) (*slackResponse, error) {
	return c.post(ctx, slackPostParams{
		channelName: channelName,
		text:        message,
		threadTS:    replytimestamp,
	})
}

// SendBlockMessage Block Kit 形式でメッセージを送信する。message.Text はフォールバックとして必須。
func (c *SlackAPIClient) SendBlockMessage(ctx context.Context, channelName gateway.SlackChannelName, message resource.SlackBlockMessage) error {
	_, err := c.post(ctx, slackPostParams{
		channelName: channelName,
		text:        message.Text,
		blocks:      message.Blocks,
		attachments: message.Attachments,
		threadTS:    message.ThreadTS,
	})
	return errors.Wrap(err, "SlackAPIClient.SendBlockMessage error")
}

// slackPostParams は chat.postMessage への1リクエスト分のパラメータ。
type slackPostParams struct {
	channelName gateway.SlackChannelName
	text        string // 必須。blocks 指定時は通知バナー/検索用フォールバックになる
	blocks      []resource.SlackBlock
	attachments []resource.SlackAttachment
	threadTS    *string
}

func (c *SlackAPIClient) post(ctx context.Context, p slackPostParams) (*slackResponse, error) {
	u, err := url.Parse(config.GetSlack().SlackBotBaseURL)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	values := url.Values{}
	values.Set("token", config.GetSlack().SlackNotificationBotToken)
	values.Set("channel", p.channelName.String())
	values.Set("text", p.text)
	if len(p.blocks) > 0 {
		b, err := json.Marshal(p.blocks)
		if err != nil {
			return nil, errors.Wrap(err, "slack blocks marshal error")
		}
		values.Set("blocks", string(b))
	}
	if len(p.attachments) > 0 {
		a, err := json.Marshal(p.attachments)
		if err != nil {
			return nil, errors.Wrap(err, "slack attachments marshal error")
		}
		values.Set("attachments", string(a))
	}
	if p.threadTS != nil {
		values.Set("thread_ts", *p.threadTS)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := c.request.GetHTTPClient()
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	if res.StatusCode != http.StatusOK {
		log.Printf("slack.sendMessage error. statusCode:%d respBody=%q blocks=%q", res.StatusCode, string(resBody), TruncateForLog(values.Get("blocks")))
		return nil, errors.New("slack.sendMessage error")
	}

	var response slackResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		return nil, errors.Wrap(err, "")
	}

	if !response.Ok {
		log.Printf("slack.sendMessage error. statusCode:%d errorMessage=%s blocks=%q", res.StatusCode, response.Error, TruncateForLog(values.Get("blocks")))
		return nil, errors.New("slack.sendMessage error")
	}

	if response.Error != "" {
		log.Printf("slack response error %+v", response.Error)
	}

	if response.Ok {
		log.Print("slack send message done.")
	}

	return &response, nil
}

// TruncateForLog はデバッグログ出力用に文字列を先頭300文字までに切り詰める。
func TruncateForLog(s string) string {
	const logPreviewMaxLen = 300
	r := []rune(s)
	if len(r) <= logPreviewMaxLen {
		return s
	}
	return string(r[:logPreviewMaxLen])
}

// 以下Redisの部分はusecaseでやるべき。
const (
	SendMessageFindTrendStockByUserLocalMethodRisingListRedisKey     string        = "SEND_MESSAGE_FIND_TREND_STOCK_BY_USERLOCAL_METHOD_RISING_LIST_REDIS_KEY"
	SendMessageFindTrendStockByUserLocalMethodFallingListRedisKey    string        = "SEND_MESSAGE_FIND_TREND_STOCK_BY_USERLOCAL_METHOD_FALLING_LIST_REDIS_KEY"
	SendMessageFindTrendStockByUserLocalMethodRisingTitleTSRedisKey  string        = "SEND_MESSAGE_FIND_TREND_STOCK_BY_USERLOCAL_METHOD_RISING_TITLE_TS_REDIS_KEY"
	SendMessageFindTrendStockByUserLocalMethodFallingTitleTSRedisKey string        = "SEND_MESSAGE_FIND_TREND_STOCK_BY_USERLOCAL_METHOD_FALLING_TITLE_TS_REDIS_KEY"
	SendMessageFindTrendStockByUserLocalMethodRisingCSVTSRedisKey    string        = "SEND_MESSAGE_FIND_TREND_STOCK_BY_USERLOCAL_METHOD_RISING_CSV_TS_REDIS_KEY"
	SendMessageFindTrendStockByUserLocalMethodFallingCSVTSRedisKey   string        = "SEND_MESSAGE_FIND_TREND_STOCK_BY_USERLOCAL_METHOD_FALLING_CSV_TS_REDIS_KEY"
	SendMessageFindTrendStockByUserLocalMethodRedisKeyTTL            time.Duration = 2 * time.Hour
	SendMessageFindTrendStockByUserLocalMethodSendedBrandRedisKeyTTL time.Duration = 30 * 24 * time.Hour // 30日
)
