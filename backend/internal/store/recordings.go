package store

import (
	"context"
	"database/sql"
	"errors"
)

type Recording struct {
	ID           int64   `json:"id"`
	SessionID    *int64  `json:"sessionId"`
	ExerciseID   *int64  `json:"exerciseId"`
	ExerciseName *string `json:"exerciseName"`
	PoseProfile  *string `json:"poseProfile"`
	Filename     string  `json:"-"`
	Mime         string  `json:"mime"`
	SizeBytes    int64   `json:"sizeBytes"`
	DurationSec  float64 `json:"durationSec"`
	Label        string  `json:"label"`
	Note         string  `json:"note"`
	CreatedAt    string  `json:"createdAt"`
}

const recordingCols = `r.id, r.session_id, r.exercise_id, e.name, e.pose_profile, r.filename, r.mime, r.size_bytes, r.duration_sec, r.label, r.note, r.created_at`
const recordingFrom = ` FROM recordings r LEFT JOIN exercises e ON e.id = r.exercise_id`

func scanRecording(row interface{ Scan(...any) error }) (Recording, error) {
	var r Recording
	err := row.Scan(&r.ID, &r.SessionID, &r.ExerciseID, &r.ExerciseName, &r.PoseProfile, &r.Filename, &r.Mime, &r.SizeBytes, &r.DurationSec, &r.Label, &r.Note, &r.CreatedAt)
	return r, err
}

// ListRecordings listet Aufnahmen, optional gefiltert nach Session.
func (s *Store) ListRecordings(ctx context.Context, sessionID *int64) ([]Recording, error) {
	q := `SELECT ` + recordingCols + recordingFrom
	args := []any{}
	if sessionID != nil {
		q += ` WHERE r.session_id = ?`
		args = append(args, *sessionID)
	}
	q += ` ORDER BY r.created_at DESC, r.id DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Recording{}
	for rows.Next() {
		r, err := scanRecording(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetRecording(ctx context.Context, id int64) (Recording, error) {
	r, err := scanRecording(s.db.QueryRowContext(ctx, `SELECT `+recordingCols+recordingFrom+` WHERE r.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

func (s *Store) CreateRecording(ctx context.Context, r Recording) (Recording, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO recordings (session_id, exercise_id, filename, mime, size_bytes, duration_sec, label, note, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.SessionID, r.ExerciseID, r.Filename, r.Mime, r.SizeBytes, r.DurationSec, r.Label, r.Note, Now())
	if err != nil {
		return r, err
	}
	id, _ := res.LastInsertId()
	return s.GetRecording(ctx, id)
}

func (s *Store) UpdateRecordingNote(ctx context.Context, id int64, note string) (Recording, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE recordings SET note = ? WHERE id = ?`, note, id)
	if err != nil {
		return Recording{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Recording{}, ErrNotFound
	}
	return s.GetRecording(ctx, id)
}

// DeleteRecording löscht den Datensatz und gibt den Dateinamen zurück,
// damit der Aufrufer die Datei entfernen kann.
func (s *Store) DeleteRecording(ctx context.Context, id int64) (string, error) {
	r, err := s.GetRecording(ctx, id)
	if err != nil {
		return "", err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM recordings WHERE id = ?`, id); err != nil {
		return "", err
	}
	return r.Filename, nil
}
