package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
)

// ensureSenderEmail ensures that sender email is set, using user email from Settings as fallback.
func ensureSenderEmail(ctx context.Context, db *queries.Base, mailSetting *models.Mail) error {
	// If sender email is already configured, no need to do anything
	if mailSetting.SenderEmail != "" {
		return nil
	}

	// Get user email from Settings as fallback
	userSettings, err := db.GetSettingByKey(ctx, "email")
	if err != nil {
		return fmt.Errorf("sender email is not configured and failed to get user email: %w", err)
	}

	userEmail, ok := userSettings["email"]
	if !ok || userEmail.Value == nil {
		return fmt.Errorf("sender email is not configured and user email is not found in settings")
	}

	userEmailStr, ok := userEmail.Value.(string)
	if !ok || userEmailStr == "" {
		return fmt.Errorf("sender email is not configured and user email is empty")
	}

	// Use user email as sender email
	mailSetting.SenderEmail = userEmailStr
	return nil
}

// SendTestLetter sends a test email letter to verify SMTP configuration.
func SendTestLetter(letterName string) error {
	db := queries.DB()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mailSetting, err := queries.GetSettingByGroup[models.Mail](ctx, db)
	if err != nil {
		return err
	}

	// Check if SMTP settings are properly configured
	if mailSetting.SMTP.Host == "" || mailSetting.SMTP.Port <= 0 || mailSetting.SMTP.Username == "" || mailSetting.SMTP.Password == "" {
		return fmt.Errorf("SMTP settings are not properly configured. Please fill in all required fields: Host, Port, Username, and Password")
	}

	// Ensure sender email is set (use user email as fallback if not configured)
	if err := ensureSenderEmail(ctx, db, mailSetting); err != nil {
		return err
	}

	settingEmail, err := db.GetSettingByKey(ctx, "email", letterName)
	if err != nil {
		return err
	}

	letter := &models.MessageMail{
		To: settingEmail["email"].Value.(string),
		Letter: models.Letter{
			Subject: "myCart test smtp settings",
			Text:    "test message",
		},
		Data: map[string]string{
			"Payment_URL":    "https://payment.com/order/1234567890",
			"Admin_Email":    "Admin Name <admin@mail.com>",
			"Site_Name":      "Site name",
			"Amount_Payment": "21.00 USD",
		},
	}

	if letterName != "smtp" {
		if err := json.Unmarshal([]byte(settingEmail[letterName].Value.(string)), &letter.Letter); err != nil {
			return err
		}
	}

	if err := SendMail(mailSetting, letter); err != nil {
		return err
	}

	return nil
}

// SendPrepaymentLetter sends an email notification before payment is completed.
func SendPrepaymentLetter(email, amountPayment, paymentURL string) error {
	db := queries.DB()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	letter, err := db.CartLetterPayment(ctx, email, amountPayment, paymentURL)
	if err != nil {
		return err
	}

	mailSetting, err := queries.GetSettingByGroup[models.Mail](ctx, db)
	if err != nil {
		return err
	}

	// Ensure sender email is set (use user email as fallback if not configured)
	if err := ensureSenderEmail(ctx, db, mailSetting); err != nil {
		return err
	}

	if err := SendMail(mailSetting, letter); err != nil {
		return err
	}

	return nil
}

// SendCartLetter sends an email notification after a cart purchase is completed.
func SendCartLetter(cartID string) error {
	db := queries.DB()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	letter, err := db.CartLetterPurchase(ctx, cartID)
	if err != nil {
		return err
	}

	mailSetting, err := queries.GetSettingByGroup[models.Mail](ctx, db)
	if err != nil {
		return err
	}

	// Ensure sender email is set (use user email as fallback if not configured)
	if err := ensureSenderEmail(ctx, db, mailSetting); err != nil {
		return err
	}

	if err := SendMail(mailSetting, letter); err != nil {
		return err
	}

	return nil
}
