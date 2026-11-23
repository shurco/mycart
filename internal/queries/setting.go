package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/security"
	"github.com/shurco/mycart/pkg/strutil"
)

// SettingQueries holds a dialect-aware handle allowing for easy querying and interaction
// with the database related to application settings.
type SettingQueries struct {
	DB *database.Conn
}

// GetSettingByGroup is a generic function that retrieves a setting from the database.
// It takes a context and a pointer to the Base struct which holds the database methods.
// The function returns a pointer to the requested setting of type T or an error if any occurs.
func GetSettingByGroup[T any](ctx context.Context, db *Base) (*T, error) {
	setting, err := db.GetSettingByGroup(ctx, new(T))
	if err != nil {
		return nil, err
	}
	return setting.(*T), nil
}

// unmarshalJSONToPointer unmarshals JSON value into a pointer of type T
func unmarshalJSONToPointer[T any](value string, ptr **T) error {
	if value == "" {
		return nil
	}
	var t T
	if err := json.Unmarshal([]byte(value), &t); err != nil {
		return err
	}
	*ptr = &t
	return nil
}

// parseSettingValue parses a string value into the appropriate type
func parseSettingValue(value string, fieldPtr any) error {
	switch ptr := fieldPtr.(type) {
	case *string:
		*ptr = value
	case *bool:
		bValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		*ptr = bValue
	case *int:
		if value == "" {
			*ptr = 0
		} else {
			iValue, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			*ptr = iValue
		}
	case **models.TruncationSettings:
		return unmarshalJSONToPointer(value, ptr)
	case **models.NumberFormatSettings:
		return unmarshalJSONToPointer(value, ptr)
	case **models.SymbolDisplaySettings:
		return unmarshalJSONToPointer(value, ptr)
	}
	return nil
}

// GetSettingByGroup retrieves settings based on the provided `settings` struct, populating it with values from the database.
func (q *SettingQueries) GetSettingByGroup(ctx context.Context, settings any) (any, error) {
	fieldMap := q.GroupFieldMap(settings)
	if fieldMap == nil {
		return nil, errors.ErrSettingNotFound
	}

	keys := make([]any, 0, len(fieldMap))
	for k := range fieldMap {
		keys = append(keys, k)
	}

	query := fmt.Sprintf("SELECT key, value FROM setting WHERE key IN (%s)", inPlaceholders(len(keys)))
	rows, err := q.DB.QueryContext(ctx, query, keys...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}

		if fieldPtr, ok := fieldMap[key]; ok {
			if err := parseSettingValue(value, fieldPtr); err != nil {
				return nil, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}

// marshalJSONFromPointer marshals a pointer of type T to JSON string
func marshalJSONFromPointer[T any](ptr *T) (string, error) {
	if ptr == nil {
		return "", nil
	}
	jsonBytes, err := json.Marshal(ptr)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// serializeSettingValue converts a field pointer to its string representation
func serializeSettingValue(valuePtr any) (string, bool, error) {
	switch v := valuePtr.(type) {
	case *string:
		return *v, true, nil
	case *bool:
		return strconv.FormatBool(*v), true, nil
	case *int:
		return strconv.Itoa(*v), true, nil
	case **models.TruncationSettings:
		value, err := marshalJSONFromPointer(*v)
		return value, true, err
	case **models.NumberFormatSettings:
		value, err := marshalJSONFromPointer(*v)
		return value, true, err
	case **models.SymbolDisplaySettings:
		value, err := marshalJSONFromPointer(*v)
		return value, true, err
	default:
		return "", false, nil
	}
}

// UpdateSettingByGroup updates the settings in the database using a transaction.
// It takes a context and a settings object of any type as arguments.
// Creates new settings if they don't exist, updates existing ones otherwise.
func (q *SettingQueries) UpdateSettingByGroup(ctx context.Context, settings any) error {
	fieldMap := q.GroupFieldMap(settings)

	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	upsertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO setting (id, key, value) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`)
	if err != nil {
		return err
	}
	defer func() { _ = upsertStmt.Close() }()

	for key, valuePtr := range fieldMap {
		value, ok, err := serializeSettingValue(valuePtr)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		newID := security.RandomString()
		if _, err = upsertStmt.ExecContext(ctx, newID, key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdatePassword updates the current user's password in the database.
func (q *SettingQueries) UpdatePassword(ctx context.Context, password *models.Password) error {
	var passwordHash string
	query := `SELECT value FROM setting WHERE key = 'password'`
	if err := q.DB.QueryRowContext(ctx, query).Scan(&passwordHash); err != nil {
		return errors.ErrUserNotFound
	}
	compareUserPassword := security.ComparePasswords(passwordHash, password.Old)
	if !compareUserPassword {
		return errors.ErrWrongPassword
	}

	hash, err := security.HashPassword(password.New)
	if err != nil {
		return err
	}

	query = `UPDATE setting SET value = ? WHERE key = 'password'`
	_, err = q.DB.ExecContext(ctx, query, hash)
	return err
}

// GetSettingByKey retrieves a setting by its key from the database.
// It accepts a context for cancellation and a string representing the key of the setting.
// Returns a pointer to a SettingName model if found, or an error if not found or any other issue occurs.
func (q *SettingQueries) GetSettingByKey(ctx context.Context, key ...string) (map[string]models.SettingName, error) {
	if len(key) == 0 {
		return nil, errors.ErrSettingNotFound
	}

	query := fmt.Sprintf("SELECT id, key, value FROM setting WHERE key IN (%s)", inPlaceholders(len(key)))
	rows, err := q.DB.QueryContext(ctx, query, strutil.ToAny(key...)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	settings := map[string]models.SettingName{}
	for rows.Next() {
		var key string
		setting := models.SettingName{}
		if err := rows.Scan(&setting.ID, &key, &setting.Value); err != nil {
			return nil, err
		}
		settings[key] = setting
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}

// UpdateSettingByKey updates the value of a setting in the database based on the provided key.
func (q *SettingQueries) UpdateSettingByKey(ctx context.Context, setting *models.SettingName) error {
	query := `UPDATE setting SET value = ? WHERE key = ? `
	_, err := q.DB.ExecContext(ctx, query, setting.Value, setting.Key)
	return err
}
