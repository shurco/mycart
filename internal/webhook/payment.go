package webhook

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
)

type Event string

const (
	PAYMENT_INITIATION Event = "payment_initiation"
	PAYMENT_CALLBACK   Event = "payment_callback"
	PAYMENT_SUCCESS    Event = "payment_success"
	PAYMENT_CANCEL     Event = "payment_cancel"
	PAYMENT_ERROR      Event = "payment_error"
)

type Payment struct {
	Event     Event `json:"event"`
	TimeStamp int64 `json:"timestamp"`
	Data      Data  `json:"data"`
}

type Data struct {
	CartID        string                `json:"cart_id,omitempty"`
	PaymentSystem litepay.PaymentSystem `json:"payment_system"`
	PaymentStatus litepay.Status        `json:"payment_status"`
	TotalAmount   int                   `json:"total_amount,omitempty"`
	Currency      string                `json:"currency,omitempty"`
	CartItems     []litepay.Item        `json:"cart_items,omitempty"`
}

// SendPaymentHook sends a payment webhook notification to the configured URL.
// Returns nil to avoid blocking the main process on webhook errors.
func SendPaymentHook(resData *Payment) error {
	db := queries.DB()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	webhookSetting, err := queries.GetSettingByGroup[models.Webhook](ctx, db)
	if err != nil {
		return err
	}

	if webhookSetting.Url == "" {
		return nil
	}

	jsonData, err := json.Marshal(resData)
	if err != nil {
		return err
	}

	res, err := Send(webhookSetting.Url, jsonData)
	if err != nil {
		logWebhookError(err, webhookSetting.Url, resData.Event, 0, nil)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(res.Body)
		logWebhookError(nil, webhookSetting.Url, resData.Event, res.StatusCode, bodyBytes)
		return nil
	}

	return nil
}

// logWebhookError logs webhook errors without blocking the main process.
func logWebhookError(err error, url string, event Event, statusCode int, bodyBytes []byte) {
	log := logging.New()
	errorLog := log.Error().
		Str("url", url).
		Str("event", string(event))

	if err != nil {
		errorLog.Err(err).Msg("payment webhook request failed")
		return
	}

	if statusCode != 0 {
		errorLog.Int("status", statusCode)
		if len(bodyBytes) > 0 {
			errorLog.Str("response", string(bodyBytes))
		}
		errorLog.Msg("payment webhook does not return 200 status")
	}
}
