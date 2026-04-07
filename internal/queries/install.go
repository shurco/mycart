package queries

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/security"
)

// InstallQueries is a struct that embeds a pointer to an sql.DB.
// This allows for the struct to have all the methods of sql.DB,
// enabling it to perform database operations directly.
type InstallQueries struct {
	*sql.DB
}

// Install performs the installation process for the cart system.
func (q *InstallQueries) Install(ctx context.Context, i *models.Install) error {
	var installed bool

	query := `SELECT value FROM setting WHERE key = 'installed'`
	if err := q.DB.QueryRowContext(ctx, query).Scan(&installed); err != nil {
		return err
	}
	if installed {
		return fmt.Errorf("%s", "Rejected because you have already installed and configured the cart")
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

	query = `UPDATE setting SET value = ? WHERE key = ?`
	stmt, err := tx.PrepareContext(ctx, query)
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
