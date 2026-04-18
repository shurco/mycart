package handlers

import (
	"github.com/shurco/mycart/internal/models"
)

// settingFactory returns a zero-valued pointer to the model that represents
// a particular setting group. Keeping this as a function (rather than a stored
// pointer) prevents handlers from sharing mutable state between requests.
type settingFactory func() any

// settingRegistry maps the setting_key URL parameter to the corresponding
// model factory. Centralising it here avoids the three nearly-identical
// `switch settingKey { ... }` blocks that previously lived in the Get/Update
// handlers — they differed only in which types they referenced, and drifting
// copies caused bugs in the past (e.g. one branch missing a new section).
//
// NOTE: "password" is intentionally excluded; it has unique validation and
// storage semantics and is handled explicitly by UpdateSetting.
var settingRegistry = map[string]settingFactory{
	"main":        func() any { return &models.Main{} },
	"social":      func() any { return &models.Social{} },
	"auth":        func() any { return &models.Auth{} },
	"jwt":         func() any { return &models.JWT{} },
	"webhook":     func() any { return &models.Webhook{} },
	"payment":     func() any { return &models.Payment{} },
	"stripe":      func() any { return &models.Stripe{} },
	"paypal":      func() any { return &models.Paypal{} },
	"spectrocoin": func() any { return &models.Spectrocoin{} },
	"coinbase":    func() any { return &models.Coinbase{} },
	"dummy":       func() any { return &models.Dummy{} },
	"mail":        func() any { return &models.Mail{} },
}

// settingModelFor returns a fresh zero-valued model for the given key, or
// nil if the key does not correspond to a known setting group.
func settingModelFor(key string) any {
	if f, ok := settingRegistry[key]; ok {
		return f()
	}
	return nil
}
