package queries

import (
	"context"
	"time"
)

// GetSession retrieves the session value for a given key if it hasn't expired.
// It takes a context and key as arguments and returns the session value and an error if any.
func (q *SettingQueries) GetSession(ctx context.Context, key string) (string, error) {
	var value string
	expires := time.Now().Unix()
	err := q.DB.QueryRowContext(ctx, `SELECT value FROM session WHERE key = ? AND expires > ?`, key, expires).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// AddSession upserts a session record by key. Using INSERT OR REPLACE makes callers
// idempotent: they can write to the same key repeatedly (e.g. to refresh a TTL-cached
// value) without first deleting the previous row.
func (q *SettingQueries) AddSession(ctx context.Context, key, value string, expires int64) error {
	_, err := q.DB.ExecContext(ctx, `INSERT OR REPLACE INTO session (key, value, expires) VALUES (?, ?, ?)`, key, value, expires)
	return err
}

// UpdateSession updates the session with a new value and expiration time for a given key.
// It takes a context, a session key, the new value to be set, and the new expiration time as arguments.
func (q *SettingQueries) UpdateSession(ctx context.Context, key, value string, expires int64) error {
	_, err := q.DB.ExecContext(ctx, `UPDATE session SET value = ?, expires = ? WHERE key = ? `, value, expires, key)
	return err
}

// DeleteSession removes a session from the database based on the provided key.
func (q *SettingQueries) DeleteSession(ctx context.Context, key string) error {
	_, err := q.DB.ExecContext(ctx, `DELETE FROM session WHERE key = ?`, key)
	return err
}
