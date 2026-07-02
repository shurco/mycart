package queries

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/security"
)

// ErrAlreadyInstalled is returned by Install if the cart has already been initialized.
var ErrAlreadyInstalled = errors.New("cart already installed")

// IsInstalled reports whether the cart has completed first-time setup.
func (q *InstallQueries) IsInstalled(ctx context.Context) (bool, error) {
	var rawInstalled string
	if err := q.DB.QueryRowContext(ctx, `SELECT value FROM setting WHERE key = 'installed'`).Scan(&rawInstalled); err != nil {
		return false, err
	}
	installed, _ := strconv.ParseBool(rawInstalled)
	return installed, nil
}

// InstallQueries is a struct that embeds a pointer to an sql.DB.
// This allows for the struct to have all the methods of sql.DB,
// enabling it to perform database operations directly.
type InstallQueries struct {
	*sql.DB
}

// Install performs the installation process for the cart system.
func (q *InstallQueries) Install(ctx context.Context, i *models.Install) error {
	installed, err := q.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		return ErrAlreadyInstalled
	}

	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	passwordHash := security.GeneratePassword(i.Password)
	jwt_secret, err := security.NewToken(passwordHash)
	if err != nil {
		return err
	}

	settings := map[string]string{
		"installed":  "true",
		"domain":     i.Domain,
		"email":      i.Email,
		"password":   passwordHash,
		"jwt_secret": jwt_secret,
	}

	stmt, err := tx.PrepareContext(ctx, `UPDATE setting SET value = ? WHERE key = ?`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for key, value := range settings {
		if _, err := stmt.ExecContext(ctx, value, key); err != nil {
			return err
		}
	}

	return tx.Commit()
}
