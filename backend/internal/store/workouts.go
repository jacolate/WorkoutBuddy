package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// Workout speichert die Blockstruktur als JSON; das Schema gehört dem Frontend
// (siehe frontend/src/lib/types.ts).
type Workout struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	Notes     string          `json:"notes"`
	Structure json.RawMessage `json:"structure"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
	// LastPlayedAt ist nur in Listen gesetzt.
	LastPlayedAt *string `json:"lastPlayedAt,omitempty"`
}

const workoutCols = `w.id, w.name, w.notes, w.structure, w.created_at, w.updated_at`

func scanWorkout(row interface{ Scan(...any) error }, extra ...any) (Workout, error) {
	var w Workout
	var structure string
	dest := append([]any{&w.ID, &w.Name, &w.Notes, &structure, &w.CreatedAt, &w.UpdatedAt}, extra...)
	err := row.Scan(dest...)
	w.Structure = json.RawMessage(structure)
	return w, err
}

func (s *Store) ListWorkouts(ctx context.Context) ([]Workout, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+workoutCols+`, (SELECT MAX(started_at) FROM sessions s WHERE s.workout_id = w.id)
		FROM workouts w
		ORDER BY COALESCE((SELECT MAX(started_at) FROM sessions s WHERE s.workout_id = w.id), w.updated_at) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Workout{}
	for rows.Next() {
		var last sql.NullString
		w, err := scanWorkout(rows, &last)
		if err != nil {
			return nil, err
		}
		if last.Valid {
			w.LastPlayedAt = &last.String
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkout(ctx context.Context, id int64) (Workout, error) {
	w, err := scanWorkout(s.db.QueryRowContext(ctx, `SELECT `+workoutCols+` FROM workouts w WHERE w.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return w, ErrNotFound
	}
	return w, err
}

func (s *Store) CreateWorkout(ctx context.Context, w Workout) (Workout, error) {
	now := Now()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO workouts (name, notes, structure, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		w.Name, w.Notes, string(w.Structure), now, now)
	if err != nil {
		return w, err
	}
	id, _ := res.LastInsertId()
	return s.GetWorkout(ctx, id)
}

func (s *Store) UpdateWorkout(ctx context.Context, w Workout) (Workout, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workouts SET name = ?, notes = ?, structure = ?, updated_at = ? WHERE id = ?`,
		w.Name, w.Notes, string(w.Structure), Now(), w.ID)
	if err != nil {
		return w, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return w, ErrNotFound
	}
	return s.GetWorkout(ctx, w.ID)
}

func (s *Store) DeleteWorkout(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM workouts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
