package queries

import (
	"github.com/shurco/mycart/internal/models"
)

// GroupFieldMap generates a map of fields based on the type of settings.
func (q *SettingQueries) GroupFieldMap(settings any) map[string]any {
	switch s := settings.(type) {
	case *models.Main:
		return mainFields(s)
	case *models.Auth:
		return authFields(s)
	case *models.JWT:
		return jwtFields(s)
	case *models.Social:
		return socialFields(s)
	case *models.Payment:
		return paymentFields(s)
	case *models.Stripe:
		return stripeFields(s)
	case *models.Paypal:
		return paypalFields(s)
	case *models.Spectrocoin:
		return spectrocoinFields(s)
	case *models.Coinbase:
		return coinbaseFields(s)
	case *models.Portone:
		return portoneFields(s)
	case *models.Dummy:
		return dummyFields(s)
	case *models.Webhook:
		return webhookFields(s)
	case *models.Mail:
		return mailFields(s)
	case *models.Branding:
		return brandingFields(s)
	case *models.Account:
		return accountFields(s)
	case *models.SEO:
		return seoFields(s)
	default:
		return nil
	}
}

// mainFields maps the keys of the Main setting group to its fields.
func mainFields(s *models.Main) map[string]any {
	return map[string]any{
		"site_name": &s.SiteName,
		"domain":    &s.Domain,
	}
}

// authFields maps the keys of the Auth setting group to its fields.
func authFields(s *models.Auth) map[string]any {
	return map[string]any{
		"email": &s.Email,
	}
}

// jwtFields maps the keys of the JWT setting group to its fields.
func jwtFields(s *models.JWT) map[string]any {
	return map[string]any{
		"jwt_secret":              &s.Secret,
		"jwt_secret_expire_hours": &s.ExpireHours,
	}
}

// socialFields maps the keys of the Social setting group to its fields.
func socialFields(s *models.Social) map[string]any {
	return map[string]any{
		"social_facebook":  &s.Facebook,
		"social_instagram": &s.Instagram,
		"social_twitter":   &s.Twitter,
		"social_dribbble":  &s.Dribbble,
		"social_github":    &s.Github,
		"social_youtube":   &s.Youtube,
		"social_other":     &s.Other,
	}
}

// paymentFields maps the keys of the Payment setting group to its fields.
func paymentFields(s *models.Payment) map[string]any {
	return map[string]any{
		"currency":       &s.Currency,
		"truncation":     &s.Truncation,
		"number_format":  &s.NumberFormat,
		"symbol_display": &s.SymbolDisplay,
	}
}

// stripeFields maps the keys of the Stripe setting group to its fields.
func stripeFields(s *models.Stripe) map[string]any {
	return map[string]any{
		"stripe_secret_key": &s.SecretKey,
		"stripe_active":     &s.Active,
	}
}

// paypalFields maps the keys of the Paypal setting group to its fields.
func paypalFields(s *models.Paypal) map[string]any {
	return map[string]any{
		"paypal_client_id":  &s.ClientID,
		"paypal_secret_key": &s.SecretKey,
		"paypal_active":     &s.Active,
	}
}

// spectrocoinFields maps the keys of the Spectrocoin setting group to its fields.
func spectrocoinFields(s *models.Spectrocoin) map[string]any {
	return map[string]any{
		"spectrocoin_merchant_id":          &s.MerchantID,
		"spectrocoin_project_id":           &s.ProjectID,
		"spectrocoin_callback_merchant_id": &s.CallbackMerchantID,
		"spectrocoin_callback_api_id":      &s.CallbackApiID,
		"spectrocoin_private_key":          &s.PrivateKey,
		"spectrocoin_active":               &s.Active,
	}
}

// coinbaseFields maps the keys of the Coinbase setting group to its fields.
func coinbaseFields(s *models.Coinbase) map[string]any {
	return map[string]any{
		"coinbase_api_key": &s.ApiKey,
		"coinbase_active":  &s.Active,
	}
}

// portoneFields maps the keys of the Portone setting group to its fields.
func portoneFields(s *models.Portone) map[string]any {
	return map[string]any{
		"portone_store_id":    &s.StoreID,
		"portone_channel_key": &s.ChannelKey,
		"portone_api_secret":  &s.ApiSecret,
		"portone_active":      &s.Active,
	}
}

// dummyFields maps the keys of the Dummy setting group to its fields.
func dummyFields(s *models.Dummy) map[string]any {
	return map[string]any{
		"dummy_active": &s.Active,
	}
}

// webhookFields maps the keys of the Webhook setting group to its fields.
func webhookFields(s *models.Webhook) map[string]any {
	return map[string]any{
		"webhook_url": &s.Url,
	}
}

// mailFields maps the keys of the Mail setting group to its fields.
func mailFields(s *models.Mail) map[string]any {
	return map[string]any{
		"mail_sender_name":  &s.SenderName,
		"mail_sender_email": &s.SenderEmail,
		"smtp_host":         &s.SMTP.Host,
		"smtp_port":         &s.SMTP.Port,
		"smtp_username":     &s.SMTP.Username,
		"smtp_password":     &s.SMTP.Password,
		"smtp_encryption":   &s.SMTP.Encryption,
	}
}

// brandingFields maps the keys of the Branding setting group to its fields.
func brandingFields(s *models.Branding) map[string]any {
	return map[string]any{
		"branding_logo":    &s.Logo,
		"branding_favicon": &s.Favicon,
		"branding_tagline": &s.Tagline,
	}
}

// accountFields maps the keys of the Account setting group to its fields.
//
// account_jwt_secret is not here on purpose: this map is what the admin
// settings read and save walks, and a key an operator never sees must
// not be rewritten with an empty value on every save. AccountSecret
// owns that row.
func accountFields(s *models.Account) map[string]any {
	return map[string]any{
		"account_enabled":          &s.Enabled,
		"account_jwt_expire_hours": &s.ExpireHours,
	}
}

// seoFields maps the keys of the SEO setting group to its fields.
func seoFields(s *models.SEO) map[string]any {
	return map[string]any{
		"seo_sitemap_enabled": &s.SitemapEnabled,
		"seo_sitemap_include": &s.SitemapInclude,
		"seo_sitemap_exclude": &s.SitemapExclude,
		"seo_llms_enabled":    &s.LlmsEnabled,
		"seo_llms_template":   &s.LlmsTemplate,
		"seo_ai_enabled":      &s.AiEnabled,
		"seo_ai_template":     &s.AiTemplate,
		"seo_robots_enabled":  &s.RobotsEnabled,
		"seo_robots_template": &s.RobotsTemplate,
		"seo_openapi_enabled": &s.OpenAPIEnabled,
		"seo_openapi_include": &s.OpenAPIInclude,
		"seo_openapi_exclude": &s.OpenAPIExclude,
		"seo_cdn_enabled":     &s.CDNEnabled,
		"seo_cdn_rules":       &s.CDNRules,
	}
}
