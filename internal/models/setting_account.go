package models

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

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
