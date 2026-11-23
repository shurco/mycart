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
