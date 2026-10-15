package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// CreateAuthSession speichert den Hash eines Session-Tokens.
func (s *Store) CreateAuthSession(ctx context.Context, tokenHash string, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO auth_sessions (token_hash, created_at, expires_at) VALUES (?, ?, ?)`,
		tokenHash, Now(), FormatTime(expires))
	return err
}

// AuthSessionExpiry liefert das Ablaufdatum einer gültigen (nicht abgelaufenen) Session.
func (s *Store) AuthSessionExpiry(ctx context.Context, tokenHash string) (time.Time, error) {
	var exp string
	err := s.db.QueryRowContext(ctx,
		`SELECT expires_at FROM auth_sessions WHERE token_hash = ? AND expires_at > ?`, tokenHash, Now()).Scan(&exp)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(TimeFormat, exp)
}

func (s *Store) ExtendAuthSession(ctx context.Context, tokenHash string, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE auth_sessions SET expires_at = ? WHERE token_hash = ?`, FormatTime(expires), tokenHash)
	return err
}

func (s *Store) DeleteAuthSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) PurgeExpiredAuthSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE expires_at <= ?`, Now())
	return err
}
