package notify

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/logx"
)

type (
	// MTS
	SMSBodyMTS struct {
		Number             string `json:"number"`      // sender number / name
		Destination        string `json:"destination"` // reciever
		Text               string `json:"text"`
		TemplateResourceID uint64 `json:"template_resource_id,omitempty"` // id of template
	}

	// Devino
	SMSBodyDevino struct {
		From     string `json:"from"`
		To       string `json:"to"`
		Text     string `json:"text"`
		Priority string `json:"priority"`
	}

	PayloadDevino struct {
		Messages []SMSBodyDevino `json:"messages"`
	}
)

const phoneRegexPattern = "(?i)^[7][9][0-9]+$"

var phonesRegex = regexp.MustCompile(phoneRegexPattern)

func SendRUSMS(req *PushNotification, cfg *config.ConfYaml) {
	if req == nil {
		return
	}
	if !cfg.SMS.Enabled {
		return
	}

	var sendSMS func(phoneNumber string, req *PushNotification, cfg config.SectionSMS) bool

	if cfg.SMS.Provider == config.SMSProviderMTS {
		sendSMS = sendViaMTS
	} else if cfg.SMS.Provider == config.SMSProviderDevino {
		sendSMS = sendViaDevino
	} else {
		logx.LogError.Errorf("Unsupported SMS provider: %s", cfg.SMS.Provider)
		return
	}

	for _, phoneNumber := range req.PhoneNumbers {
		if !sendSMS(phoneNumber, req, cfg.SMS) {
			return
		}
	}
}

func sendViaMTS(phoneNumber string, req *PushNotification, cfg config.SectionSMS) bool {
	phoneNumber = strings.ReplaceAll(phoneNumber, "+", "")

	if !phonesRegex.MatchString(phoneNumber) {
		logx.LogAccess.Debugf("SMS skipping phone number %s, doesn't match pattern: %s",
			phoneNumber, phoneRegexPattern)
		return true
	}

	var (
		templateID uint64
		err        error
	)

	if req.TemplateID != "" {
		templateID, err = strconv.ParseUint(req.TemplateID, 10, 64)
		if err != nil {
			logx.LogError.Errorf("SMS skipping phone number %s, invalid template id: %s", phoneNumber, req.TemplateID)
			return true
		}
	}

	payload := SMSBodyMTS{
		Number:             cfg.MTSSenderNumber,
		Destination:        phoneNumber,
		Text:               req.SMSMessage,
		TemplateResourceID: templateID,
	}

	authKey := fmt.Sprintf("Bearer %s", cfg.MTSApiKey)
	return sendSMS(cfg.MTSApiUrl, authKey, phoneNumber, payload)
}

func sendViaDevino(phoneNumber string, req *PushNotification, cfg config.SectionSMS) bool {
	payload := PayloadDevino{
		Messages: []SMSBodyDevino{
			{
				From:     cfg.DevinoSenderNumber,
				To:       phoneNumber,
				Text:     req.SMSMessage,
				Priority: "HIGH",
			},
		},
	}

	authKey := fmt.Sprintf("Key %s", cfg.DevinoApiKey)
	return sendSMS(cfg.DevinoApiUrl, authKey, phoneNumber, payload)
}

func sendSMS(url, authKey, phoneNumber string, payload any) bool {
	logx.LogAccess.Debugf("Start push notification via SMS, url: %s", url)

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		logx.LogError.Errorf("error sending SMS to: %s, err: %v", phoneNumber, err)
		return false
	}

	body := bytes.NewBuffer(jsonBody)
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		logx.LogError.Errorf("error sending SMS to: %s, err: %v", phoneNumber, err)
		return false
	}

	request.Header.Set("Authorization", authKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		logx.LogError.Error(err)
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			logx.LogError.Error(err)
			return false
		}
		logx.LogAccess.Debugf("SMS response status code != 200, response body: %s", string(body))
		return false
	}
	return true
}
