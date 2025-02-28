package notify

import (
	"bytes"
	"io"
	"net/http"

	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/logx"
)

type TelphinCallRequest struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Number    string `json:"number"`
	AuthCode  string `json:"auth_code"`
}

// TODO: remake logs as in send via FCM, APNS, HMS
func SendTelphinCall(req *PushNotification, cfg *config.ConfYaml) {
	if req == nil || !cfg.CallAuto.Enabled {
		return
	}

	for _, phoneNumber := range req.PhoneNumbers {
		logx.LogAccess.Debugf("| TELPHIN CALL | PHONE NUMBER: %s | MESSAGE: %s",
			hideString(phoneNumber, 3), req.SMSMessage)

		bodyModel := TelphinCallRequest{
			AppID:     cfg.CallAuto.AppID,
			AppSecret: cfg.CallAuto.AppSecret,
			Number:    phoneNumber,
			AuthCode:  req.SMSMessage,
		}

		bodyBytes, _ := json.Marshal(bodyModel)

		request, err := http.NewRequest(http.MethodPost, cfg.CallAuto.ApiURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			logx.LogAccess.Errorf("| TELPHIN CALL ERROR | ERROR: %v", err)
			return
		}

		request.Header.Set("content-type", "application/json")

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			logx.LogAccess.Errorf("| TELPHIN CALL ERROR | ERROR: %v | RESPONSE: %v", err, response)
			return
		}
		defer response.Body.Close()

		respBodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			logx.LogAccess.Errorf("| TELPHIN CALL ERROR | ERROR: %v", err)
			return
		}

		logx.LogAccess.Debugf("TELPHIN RESPONSE - BODY: %s; STATUS CODE: %d", string(respBodyBytes), response.StatusCode)
	}
}
