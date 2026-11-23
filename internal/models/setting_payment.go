package models

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

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
