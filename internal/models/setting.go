package models

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Main is ...
type Main struct {
	SiteName string `json:"site_name"`
	Domain   string `json:"domain"`
	Email    string `json:"email"`
}

// Validate is ...
func (v Main) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.SiteName, validation.Required, validation.Length(1, 100)),
		validation.Field(&v.Domain, is.Domain),
		validation.Field(&v.Email, is.Email),
	)
}

// Auth is ...
type Auth struct {
	Email string `json:"email"`
	// auth providers
}

// Validate is ...
func (v Auth) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Email, is.Email),
	)
}

// Password is ..
type Password struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// Validate is ...
func (v Password) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Old, validation.Length(6, 72)),
		validation.Field(&v.New, validation.Length(6, 72)),
	)
}

// Payment is ...
type Payment struct {
	Currency      string                 `json:"currency"`
	Truncation    *TruncationSettings    `json:"truncation,omitempty"`
	NumberFormat  *NumberFormatSettings  `json:"number_format,omitempty"`
	SymbolDisplay *SymbolDisplaySettings `json:"symbol_display,omitempty"`
}

// Validate is ...
func (v Payment) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Currency, is.CurrencyCode),
		validation.Field(&v.Truncation, validation.By(validateTruncation)),
		validation.Field(&v.NumberFormat, validation.By(validateNumberFormat)),
		validation.Field(&v.SymbolDisplay, validation.By(validateSymbolDisplay)),
	)
}

// validateTruncation validates truncation settings
func validateTruncation(value interface{}) error {
	if value == nil {
		return nil // truncation is optional
	}

	truncation, ok := value.(*TruncationSettings)
	if !ok || truncation == nil {
		return nil
	}

	validModes := map[string]bool{"none": true, "fixed": true, "flexible": true}

	// Validate admin settings
	for currency, settings := range truncation.Admin {
		if !validModes[settings.Mode] {
			return validation.NewError("truncation_invalid_mode",
				"mode must be 'none', 'fixed', or 'flexible' for "+currency)
		}
		if settings.Mode == "fixed" && settings.FixedUnit == "" {
			return validation.NewError("truncation_missing_unit",
				"fixed_unit required when mode is 'fixed' for "+currency)
		}
	}

	// Validate storefront settings
	for currency, settings := range truncation.Storefront {
		if !validModes[settings.Mode] {
			return validation.NewError("truncation_invalid_mode",
				"mode must be 'none', 'fixed', or 'flexible' for "+currency)
		}
		if settings.Mode == "fixed" && settings.FixedUnit == "" {
			return validation.NewError("truncation_missing_unit",
				"fixed_unit required when mode is 'fixed' for "+currency)
		}
	}

	return nil
}

// validateNumberFormat validates number format settings
func validateNumberFormat(value interface{}) error {
	if value == nil {
		return nil // number_format is optional
	}

	nf, ok := value.(*NumberFormatSettings)
	if !ok || nf == nil {
		return nil
	}

	if nf.DecimalPrecision < 0 || nf.DecimalPrecision > 2 {
		return validation.NewError("number_format_invalid_precision",
			"decimal_precision must be 0, 1, or 2")
	}

	return nil
}

// validateSymbolDisplay validates symbol display settings
func validateSymbolDisplay(value interface{}) error {
	if value == nil {
		return nil // symbol_display is optional
	}

	sd, ok := value.(*SymbolDisplaySettings)
	if !ok || sd == nil {
		return nil
	}

	validModes := map[string]bool{"currency": true, "language": true}

	if sd.Admin != "" && !validModes[sd.Admin] {
		return validation.NewError("symbol_display_invalid_mode",
			"admin symbol_display must be 'currency' or 'language'")
	}

	if sd.Storefront != "" && !validModes[sd.Storefront] {
		return validation.NewError("symbol_display_invalid_mode",
			"storefront symbol_display must be 'currency' or 'language'")
	}

	return nil
}

// CurrencyTruncationSettings defines truncation mode for a currency
type CurrencyTruncationSettings struct {
	Mode      string `json:"mode"`                 // "none", "fixed", or "flexible"
	FixedUnit string `json:"fixed_unit,omitempty"` // e.g., "K", "M", "만", "천"
}

// NumberFormatSettings defines global number formatting options
type NumberFormatSettings struct {
	DecimalPrecision  int  `json:"decimal_precision"`   // 0, 1, or 2
	ShowTrailingZeros bool `json:"show_trailing_zeros"` // true or false
}

// SymbolDisplaySettings defines currency display mode per context
type SymbolDisplaySettings struct {
	Admin      string `json:"admin"`      // "currency" or "language"
	Storefront string `json:"storefront"` // "currency" or "language"
}

// TruncationSettings holds admin and storefront truncation configs
type TruncationSettings struct {
	Admin      map[string]CurrencyTruncationSettings `json:"admin"`
	Storefront map[string]CurrencyTruncationSettings `json:"storefront"`
}

// Stripe is ...
type Stripe struct {
	SecretKey string `json:"secret_key"`
	Active    bool   `json:"active"`
}

// Validate is ...
func (v Stripe) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.SecretKey, validation.Length(100, 130)),
	)
}

// Paypal is ...
type Paypal struct {
	ClientID  string `json:"client_id"`
	SecretKey string `json:"secret_key"`
	Active    bool   `json:"active"`
}

// Validate is ...
func (v Paypal) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ClientID, validation.Length(80, 80)),
		validation.Field(&v.SecretKey, validation.Length(80, 80)),
	)
}

// Spectrocoin is ...
type Spectrocoin struct {
	MerchantID string `json:"merchant_id"`
	ProjectID  string `json:"project_id"`
	// CallbackMerchantID and CallbackApiID are the numeric merchantId and
	// apiId SpectroCoin reports in every order callback. Of all the identity
	// fields a callback carries they are the only two that both name this
	// shop's merchant account and are covered by SpectroCoin's signature, so
	// the callback handler compares them against these values before trusting
	// anything else in the payload. The UUIDs above travel outside the
	// signature and cannot serve that purpose: anyone may set them.
	//
	// Zero means unconfigured, and the callback handler refuses to match it
	// rather than treating an absent setting as a wildcard. Zero is allowed
	// here because this struct cannot tell "a shop that does not use
	// SpectroCoin" from "an operator who has not filled the values in yet",
	// and a shop that does not use it must still be able to save its settings.
	CallbackMerchantID int    `json:"callback_merchant_id"`
	CallbackApiID      int    `json:"callback_api_id"`
	PrivateKey         string `json:"private_key"`
	Active             bool   `json:"active"`
}

// Validate is ...
func (v Spectrocoin) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.MerchantID, is.UUID),
		validation.Field(&v.ProjectID, is.UUID),
		validation.Field(&v.PrivateKey, validation.Length(1700, 2200)),
	)
}

// Coinbase is ...
type Coinbase struct {
	ApiKey string `json:"api_key"`
	Active bool   `json:"active"`
}

// Validate is ...
func (v Coinbase) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ApiKey, validation.Length(20, 200)),
	)
}

// Portone is ...
type Portone struct {
	StoreID             string   `json:"store_id"`
	ChannelKey          string   `json:"channel_key"`
	ApiSecret           string   `json:"api_secret"`
	Active              bool     `json:"active"`
	DebugEnabled        bool     `json:"debug_enabled"`
	SupportedCurrencies []string `json:"supported_currencies"`
}

// Validate is ...
func (v Portone) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.StoreID, validation.Length(24, 50)),
		validation.Field(&v.ChannelKey, validation.Length(20, 100)),
		validation.Field(&v.ApiSecret, validation.Length(30, 200)),
	)
}

// Dummy is ...
type Dummy struct {
	Active bool `json:"active"`
}

// PaymentSystem is ...
type PaymentSystem struct {
	Active      []string    `json:"active"`
	Stripe      Stripe      `json:"stripe"`
	Paypal      Paypal      `json:"paypal"`
	Spectrocoin Spectrocoin `json:"spectrocoin"`
	Coinbase    Coinbase    `json:"coinbase"`
	Portone     Portone     `json:"portone"`
	Dummy       Dummy       `json:"dummy"`
}

// Validate is ...
func (v PaymentSystem) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Stripe),
		validation.Field(&v.Paypal),
		validation.Field(&v.Spectrocoin),
		validation.Field(&v.Coinbase),
		validation.Field(&v.Portone),
	)
}

type Webhook struct {
	Url string `json:"url"`
}

// Validate is ...
func (v Webhook) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Url, is.URL))
}

type Social struct {
	Facebook  string `json:"facebook,omitempty"`
	Instagram string `json:"instagram,omitempty"`
	Twitter   string `json:"twitter,omitempty"`
	Dribbble  string `json:"dribbble,omitempty"`
	Github    string `json:"github,omitempty"`
	Youtube   string `json:"youtube,omitempty"`
	Other     string `json:"other,omitempty"`
}

// Validate is ...
func (v Social) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Facebook, validation.Length(3, 20)),
		validation.Field(&v.Instagram, validation.Length(3, 20)),
		validation.Field(&v.Twitter, validation.Length(3, 20)),
		validation.Field(&v.Github, validation.Length(3, 20)),
		validation.Field(&v.Youtube, validation.Length(3, 20)),
		validation.Field(&v.Other, is.URL),
	)
}

// SettingName is ...
type SettingName struct {
	ID    string `json:"id,omitempty"`
	Key   string `json:"key"`
	Value any    `json:"value,omitempty"`
}

// Validate is ...
func (v SettingName) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ID, validation.Length(15, 15)),
		validation.Field(&v.Key, validation.Required),
	)
}

// Mail is ...
type Mail struct {
	SenderName  string `json:"sender_name"`
	SenderEmail string `json:"sender_email"`
	SMTP        SMTP   `json:"smtp"`
}

// Validate is ...
func (v Mail) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.SenderName, validation.Length(2, 30)),
		validation.Field(&v.SenderEmail, is.Email),
		validation.Field(&v.SMTP),
	)
}

// Letter ...
type Letter struct {
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Html    string `json:"html"`
}

// Validate is ...
func (v Letter) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Subject, validation.Length(5, 255)),
	)
}

// SMTP is ...
type SMTP struct {
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	Encryption string `json:"encryption,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

// Validate is ...
func (v SMTP) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Host, is.Host),
		validation.Field(&v.Port, validation.Required, validation.Min(1), validation.Max(65535)),
		// validation.Field(&v.Encryption),
		validation.Field(&v.Username, validation.Length(3, 100)),
		validation.Field(&v.Password, validation.Length(3, 100)),
	)
}

// MessageMail ...
type MessageMail struct {
	To     string            `json:"to"`
	Letter Letter            `json:"letter"`
	Data   map[string]string `json:"data"`
	Files  []File            `json:"files,omitempty"`
}

// Validate is ...
func (v MessageMail) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.To, is.Email),
	)
}

// Account holds the storefront customer-cabinet settings.
//
// The cabinet is off on a fresh installation (see the account_enabled row in
// migrations/20260913000000_customer_account.sql) so that a shop that only
// needs a guest checkout is not handed a sign-in form it never asked for.
// The signing key for storefront sessions is deliberately not a field here.
// GroupFieldMap is what the admin read/save path walks: a field would be
// written back on every save, so an operator toggling the cabinet on would
// blank a key they were never shown, invalidating every signed-in customer.
// queries.CustomerQueries.AccountSecret owns that key instead.
type Account struct {
	Enabled bool `json:"enabled"`
	// ExpireHours is how long a cabinet session lasts. Zero falls back to
	// queries.DefaultAccountExpireHours.
	ExpireHours int `json:"expire_hours"`
}

// Validate is ...
func (v Account) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ExpireHours, validation.Min(0), validation.Max(8760)),
	)
}

// Branding holds the shop's own marks: the logo the storefront draws in its
// header, the favicon it hands the browser, and an optional line printed beside
// the logo.
//
// Both files are addressed by the name they were stored under in lc_uploads
// rather than by a path or a URL, so a value here cannot point the storefront at
// a file outside the shop — the site builds "/uploads/<value>" from it. The name
// is the store's and not the operator's: a UUID and an extension, which is why
// the extension rides along in the value while the original file name is kept
// nowhere in this group.
type Branding struct {
	// The three fields are always sent, empty strings included. A group that
	// dropped its empty members would answer a shop with a blank tagline with
	// {}, and a client binding a field to that gets undefined rather than a
	// string — which is a crash, not an empty field.
	Logo    string `json:"logo"`
	Favicon string `json:"favicon"`
	// Tagline is printed beside the logo. Empty leaves the header with the mark
	// alone, which is how a shop that has uploaded nothing is drawn.
	Tagline string `json:"tagline"`
}

// Validate is ...
func (v Branding) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Tagline, validation.Length(0, 120)),
		// Both are optional, so the length floor is zero: a shop that has
		// uploaded nothing keeps the built-in mark, and clearing the field is
		// how it goes back to it.
		validation.Field(&v.Logo, validation.Length(0, 100), validation.By(validateStoredFileName)),
		validation.Field(&v.Favicon, validation.Length(0, 100), validation.By(validateStoredFileName)),
	)
}

// validateStoredFileName rejects a value that could name something other than a
// file this shop stored in lc_uploads. The storefront turns the value into
// "/uploads/<value>", so a separator or a traversal here would be a path out of
// the uploads directory. The upload handler writes UUIDs and never produces one;
// this is the check that holds when the group is written by something else.
func validateStoredFileName(value any) error {
	name, ok := value.(string)
	if !ok || name == "" {
		return nil
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return validation.NewError("branding_invalid_file", "must be a file name, not a path")
	}
	return nil
}
