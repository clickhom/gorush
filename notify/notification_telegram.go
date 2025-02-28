package notify

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/logx"
)

type (
	telegramGatewayRequest struct {
		PhoneNumber string `json:"phone_number"`
		Code        string `json:"code"`
		CallbackURL string `json:"callback_url"`
		TTL         int    `json:"ttl"`
	}

	telegramGatewayResponse struct {
		OK     bool   `json:"ok"`
		Error  string `json:"error"`
		Result struct {
			RequestID string `json:"request_id"`
		} `json:"result"`
	}
)

// TODO: remake logs as in send via FCM, APNS, HMS
func SendTelegramGateway(req *PushNotification, cfg *config.ConfYaml) bool {
	if req == nil || !cfg.TelegramGateway.Enabled {
		return false
	}

	for i, phoneNumber := range req.PhoneNumbers {
		if requestID, ok := sendTelegramGateway(cfg, phoneNumber, req.TelegramGatewayCode); !ok {
			SendRUSMS(req, cfg, i)
		} else {
			sendAt := time.Now().Add(10 * time.Second).Unix()
			scheduleRUSMS(requestID, sendAt, req, cfg, i) // going to be canceled on telegram delivered event
		}
	}

	return true
}

func sendTelegramGateway(cfg *config.ConfYaml, phoneNumber, code string) (string, bool) {
	logx.LogAccess.Debugf("Start Telegram gateway push, phone number: %s", hideString(phoneNumber, 3))

	reqBodyBytes, _ := json.Marshal(telegramGatewayRequest{
		PhoneNumber: phoneNumber,
		Code:        code,
		CallbackURL: cfg.TelegramGateway.CallbackURL,
		TTL:         60,
	})

	req, err := http.NewRequest(http.MethodPost, cfg.TelegramGateway.ApiURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		logx.LogError.Error(err)
		return "", false
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.TelegramGateway.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		logx.LogError.Error(err)
		return "", false
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logx.LogError.Error(err)
		return "", false
	}

	if resp.StatusCode != http.StatusOK {
		logx.LogAccess.Debugf("Telegram gateway response status code != 200, response body: %s", string(respBodyBytes))
		return "", false
	}

	var respBody telegramGatewayResponse
	if err := json.Unmarshal(respBodyBytes, &respBody); err != nil {
		logx.LogError.Error(err)
		return "", false
	}

	if !respBody.OK {
		logx.LogAccess.Debugf("Telegram gateway response is not ok, response body: %s", string(respBodyBytes))
		return respBody.Result.RequestID, false
	}

	return respBody.Result.RequestID, true
}
