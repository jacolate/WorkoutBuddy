package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// Session ist ein durchgeführtes (oder laufendes) Training.
type Session struct {
	ID             int64           `json:"id"`
	WorkoutID      *int64          `json:"workoutId"`
	WorkoutName    string          `json:"workoutName"`
	StartedAt      string          `json:"startedAt"`
	FinishedAt     *string         `json:"finishedAt"`
	Snapshot       json.RawMessage `json:"snapshot"`
	Log            json.RawMessage `json:"log"`
	Notes          string          `json:"notes"`
	RecordingCount int             `json:"recordingCount"`
}

const sessionCols = `s.id, s.workout_id, s.workout_name, s.started_at, s.finished_at, s.snapshot, s.log, s.notes,
	(SELECT COUNT(*) FROM recordings r WHERE r.session_id = s.id)`

func scanSession(row interface{ Scan(...any) error }) (Session, error) {
	var se Session
	var snapshot, log string
	err := row.Scan(&se.ID, &se.WorkoutID, &se.WorkoutName, &se.StartedAt, &se.FinishedAt, &snapshot, &log, &se.Notes, &se.RecordingCount)
	se.Snapshot = json.RawMessage(snapshot)
	se.Log = json.RawMessage(log)
	return se, err
}

func (s *Store) ListSessions(ctx context.Context, limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+sessionCols+` FROM sessions s ORDER BY s.started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		se, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, se)
	}
	return out, rows.Err()
}

func (s *Store) GetSession(ctx context.Context, id int64) (Session, error) {
	se, err := scanSession(s.db.QueryRowContext(ctx, `SELECT `+sessionCols+` FROM sessions s WHERE s.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return se, ErrNotFound
	}
	return se, err
}

func (s *Store) CreateSession(ctx context.Context, se Session) (Session, error) {
	if se.StartedAt == "" {
		se.StartedAt = Now()
	}
	if len(se.Log) == 0 {
		se.Log = json.RawMessage("[]")
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (workout_id, workout_name, started_at, finished_at, snapshot, log, notes) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		se.WorkoutID, se.WorkoutName, se.StartedAt, se.FinishedAt, string(se.Snapshot), string(se.Log), se.Notes)
	if err != nil {
		return se, err
	}
	id, _ := res.LastInsertId()
	return s.GetSession(ctx, id)
}

// SessionPatch enthält die änderbaren Felder; nil bedeutet „unverändert“.
type SessionPatch struct {
	FinishedAt *string          `json:"finishedAt"`
	Log        *json.RawMessage `json:"log"`
	Notes      *string          `json:"notes"`
}

func (s *Store) PatchSession(ctx context.Context, id int64, p SessionPatch) (Session, error) {
	se, err := s.GetSession(ctx, id)
	if err != nil {
		return se, err
	}
	if p.FinishedAt != nil {
		se.FinishedAt = p.FinishedAt
	}
	if p.Log != nil {
		se.Log = *p.Log
	}
	if p.Notes != nil {
		se.Notes = *p.Notes
	}
	_, err = s.db.ExecContext(ctx, `UPDATE sessions SET finished_at = ?, log = ?, notes = ? WHERE id = ?`,
		se.FinishedAt, string(se.Log), se.Notes, id)
	if err != nil {
		return se, err
	}
	return s.GetSession(ctx, id)
}

func (s *Store) DeleteSession(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
