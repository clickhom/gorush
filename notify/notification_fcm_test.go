package notify

import (
	"context"
	"os"
	"reflect"
	"testing"

	"firebase.google.com/go/v4/messaging"
	"github.com/appleboy/gorush/config"
	"github.com/appleboy/gorush/core"

	"github.com/stretchr/testify/assert"
)

func TestMissingAndroidCredential(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = ""

	err := CheckPushConf(cfg)

	assert.Error(t, err)
	assert.Equal(t, "missing fcm credential data", err.Error())
}

func TestMissingKeyForInitFCMClient(t *testing.T) {
	cfg, _ := config.LoadConf()
	cfg.Android.Credential = ""
	cfg.Android.KeyPath = ""
	client, err := InitFCMClient(context.Background(), cfg)

	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Equal(t, "missing fcm credential data", err.Error())
}

func TestPushToAndroidWrongToken(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	req := &PushNotification{
		Tokens:   []string{"aaaaaa", "bbbbb"},
		Platform: core.PlatformAndroid,
		Message:  "Welcome",
	}

	// Android Success count: 0, Failure count: 2
	resp, err := PushToAndroid(context.Background(), req, cfg)
	assert.Nil(t, err)
	assert.Len(t, resp.Logs, 2)
}

func TestPushToAndroidRightTokenForJSONLog(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")
	// log for json
	cfg.Log.Format = "json"

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := &PushNotification{
		Tokens:   []string{androidToken},
		Platform: core.PlatformAndroid,
		Message:  "Welcome",
	}

	resp, err := PushToAndroid(context.Background(), req, cfg)
	assert.Nil(t, err)
	assert.Len(t, resp.Logs, 0)
}

func TestPushToAndroidRightTokenForStringLog(t *testing.T) {
	cfg, _ := config.LoadConf()

	cfg.Android.Enabled = true
	cfg.Android.Credential = os.Getenv("FCM_CREDENTIAL")

	androidToken := os.Getenv("FCM_TEST_TOKEN")

	req := &PushNotification{
		Tokens:   []string{androidToken},
		Platform: core.PlatformAndroid,
		Message:  "Welcome",
	}

	resp, err := PushToAndroid(context.Background(), req, cfg)
	assert.Nil(t, err)
	assert.Len(t, resp.Logs, 0)
}

func TestFCMMessage(t *testing.T) {
	var err error

	// the message must specify at least one registration ID
	req := &PushNotification{
		Message: "Test",
		Tokens:  []string{},
	}

	err = CheckMessage(req)
	assert.Error(t, err)

	// ignore check token length if send topic message
	req = &PushNotification{
		Message:  "Test",
		Platform: core.PlatformAndroid,
		Topic:    "/topics/foo-bar",
	}

	err = CheckMessage(req)
	assert.NoError(t, err)

	// "condition": "'dogs' in topics || 'cats' in topics",
	req = &PushNotification{
		Message:   "Test",
		Platform:  core.PlatformAndroid,
		Condition: "'dogs' in topics || 'cats' in topics",
	}

	err = CheckMessage(req)
	assert.NoError(t, err)

	// the message may specify at most 1000 registration IDs
	req = &PushNotification{
		Message:  "Test",
		Platform: core.PlatformAndroid,
		Tokens:   make([]string, 501),
	}

	err = CheckMessage(req)
	assert.Error(t, err)

	// Pass
	req = &PushNotification{
		Message:  "Test",
		Platform: core.PlatformAndroid,
		Tokens:   []string{"XXXXXXXXX"},
	}

	err = CheckMessage(req)
	assert.NoError(t, err)
}

func TestAndroidNotificationStructure(t *testing.T) {
	test := "test"
	req := &PushNotification{
		Tokens:         []string{"a", "b"},
		Message:        "Welcome",
		To:             test,
		Priority:       HIGH,
		MutableContent: true,
		Title:          test,
		Sound:          test,
		Data: D{
			"a": "1",
			"b": 2,
			"json": map[string]interface{}{
				"c": "3",
				"d": 4,
			},
		},
		Notification: &messaging.Notification{
			Title: test,
			Body:  "",
		},
	}

	messages := GetAndroidNotification(req)

	assert.Equal(t, test, messages[0].Notification.Title)
	assert.Equal(t, "Welcome", messages[0].Notification.Body)
	assert.Equal(t, "1", messages[0].Data["a"])
	assert.Equal(t, "2", messages[0].Data["b"])
	assert.Equal(t, "{\"c\":\"3\",\"d\":4}", messages[0].Data["json"])
	assert.NotNil(t, messages[0].APNS)
	assert.Equal(t, req.Sound, messages[0].APNS.Payload.Aps.Sound)
	assert.Equal(t, req.MutableContent, messages[0].APNS.Payload.Aps.MutableContent)

	// test empty body
	req = &PushNotification{
		Tokens: []string{"a", "b"},
		To:     test,
		Notification: &messaging.Notification{
			Body: "",
		},
	}
	messages = GetAndroidNotification(req)

	assert.Equal(t, "", messages[0].Notification.Body)
}

func TestAndroidBackgroundNotificationStructure(t *testing.T) {
	data := map[string]any{
		"a": "1",
		"b": 2,
		"json": map[string]interface{}{
			"c": "3",
			"d": 4,
		},
	}
	req := &PushNotification{
		Tokens:           []string{"a", "b"},
		Priority:         HIGH,
		ContentAvailable: true,
		Data:             data,
	}

	messages := GetAndroidNotification(req)

	assert.Equal(t, "1", messages[0].Data["a"])
	assert.Equal(t, "2", messages[0].Data["b"])
	assert.Equal(t, "{\"c\":\"3\",\"d\":4}", messages[0].Data["json"])
	assert.NotNil(t, messages[0].APNS)
	assert.Equal(t, req.ContentAvailable, messages[0].APNS.Payload.Aps.ContentAvailable)
	assert.True(t, reflect.DeepEqual(data, messages[0].APNS.Payload.Aps.CustomData))
}
