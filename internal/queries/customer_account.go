package queries

import (
	"context"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/security"
)

// Account returns the storefront account settings.
func (q *CustomerQueries) Account(ctx context.Context) (*models.Account, error) {
	setting, err := NewBase(q.DB).GetSettingByGroup(ctx, new(models.Account))
	if err != nil {
		return nil, err
	}
	return setting.(*models.Account), nil
}

// AccountEnabled reports whether the customer cabinet is available. It is FALSE
// on a fresh installation.
func (q *CustomerQueries) AccountEnabled(ctx context.Context) (bool, error) {
	account, err := q.Account(ctx)
	if err != nil {
		return false, err
	}
	return account.Enabled, nil
}

// AccountSecret returns the key that signs storefront session tokens, creating
// it on first use.
//
// It is read and written only from here: the key is not a field of
// models.Account, so it is neither returned to the admin API nor rewritten when
// an operator saves the cabinet settings.
//
// It must not be the admin jwt_secret. Both keys sign tokens whose only claim
// is a session id, and both middlewares accept a token on the strength of a
// row in the shared session table — so a single key, plus one storefront
// session, would be enough to sign a token the admin API would honour. The
// secret is generated here rather than seeded by a migration because a key
// written into a public SQL file is a published key.
func (q *CustomerQueries) AccountSecret(ctx context.Context) (string, error) {
	var value string
	if err := q.DB.QueryRowContext(ctx,
		`SELECT value FROM setting WHERE key = ?`, accountSecretKey).Scan(&value); err != nil {
		return "", err
	}
	if value != "" {
		return value, nil
	}

	token, err := security.NewToken(security.RandomString())
	if err != nil {
		return "", err
	}

	// Fill the empty row, then re-read it. Two requests that find the secret
	// missing at the same moment must end up agreeing on one key: the loser of
	// this race returns what was stored, not what it generated, or it would
	// hand out tokens that nothing can verify.
	if _, err := q.DB.ExecContext(ctx,
		`UPDATE setting SET value = ? WHERE key = ? AND value = ''`, token, accountSecretKey); err != nil {
		return "", err
	}

	if err := q.DB.QueryRowContext(ctx,
		`SELECT value FROM setting WHERE key = ?`, accountSecretKey).Scan(&value); err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.ErrSettingNotFound
	}

	return value, nil
}
