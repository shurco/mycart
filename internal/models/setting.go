package models

import (
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
		validation.Field(&v.SiteName, validation.Min(6)),
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
	Currency string `json:"currency"`
}

// Validate is ...
func (v Payment) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Currency, is.CurrencyCode),
	)
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
	PrivateKey string `json:"private_key"`
	Active     bool   `json:"active"`
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
	Dummy       Dummy       `json:"dummy"`
}

// Validate is ...
func (v PaymentSystem) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Stripe),
		validation.Field(&v.Paypal),
		validation.Field(&v.Spectrocoin),
		validation.Field(&v.Coinbase),
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
		validation.Field(&v.Username, validation.Length(3, 20)),
		validation.Field(&v.Password, validation.Length(3, 20)),
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
