package queries

import (
	"context"
	"database/sql"

	"github.com/shurco/mycart/pkg/errors"
)

// AuthQueries is a struct that embeds *sql.DB to provide database functionality.
// This structure can be used to create methods that will execute SQL queries related to authentication.
type AuthQueries struct {
	*sql.DB
}

// GetPasswordByEmail retrieves the password for a user by their email.
func (q *AuthQueries) GetPasswordByEmail(ctx context.Context, email string) (string, error) {
	query := `SELECT key, value FROM setting WHERE key IN ('email', 'password')`
	rows, err := q.DB.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, value string
		err := rows.Scan(&key, &value)
		if err != nil {
			return "", err
		}

		switch key {
		case "email":
			if value != email {
				return "", errors.ErrUserEmailNotFound
			}
		case "password":
			if value == "" {
				return "", errors.ErrUserPasswordNotFound
			}
			return value, nil
		}
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	return "", errors.ErrUserNotFound
}
