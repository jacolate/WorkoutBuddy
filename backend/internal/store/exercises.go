package store

import (
	"context"
	"database/sql"
	"errors"
)

type Exercise struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Unilateral      bool     `json:"unilateral"`
	DefaultWeightKg *float64 `json:"defaultWeightKg"`
	PoseProfile     *string  `json:"poseProfile"`
	Cues            string   `json:"cues"`
	CameraHint      string   `json:"cameraHint"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

const exerciseCols = `id, name, description, category, unilateral, default_weight_kg, pose_profile, cues, camera_hint, created_at, updated_at`

func scanExercise(row interface{ Scan(...any) error }) (Exercise, error) {
	var e Exercise
	var uni int
	err := row.Scan(&e.ID, &e.Name, &e.Description, &e.Category, &uni, &e.DefaultWeightKg, &e.PoseProfile, &e.Cues, &e.CameraHint, &e.CreatedAt, &e.UpdatedAt)
	e.Unilateral = uni != 0
	return e, err
}

func (s *Store) ListExercises(ctx context.Context) ([]Exercise, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+exerciseCols+` FROM exercises ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Exercise{}
	for rows.Next() {
		e, err := scanExercise(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetExercise(ctx context.Context, id int64) (Exercise, error) {
	e, err := scanExercise(s.db.QueryRowContext(ctx, `SELECT `+exerciseCols+` FROM exercises WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

func (s *Store) CreateExercise(ctx context.Context, e Exercise) (Exercise, error) {
	now := Now()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO exercises (name, description, category, unilateral, default_weight_kg, pose_profile, cues, camera_hint, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Name, e.Description, e.Category, boolToInt(e.Unilateral), e.DefaultWeightKg, e.PoseProfile, e.Cues, e.CameraHint, now, now)
	if err != nil {
		return e, err
	}
	id, _ := res.LastInsertId()
	return s.GetExercise(ctx, id)
}

func (s *Store) UpdateExercise(ctx context.Context, e Exercise) (Exercise, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE exercises SET name = ?, description = ?, category = ?, unilateral = ?, default_weight_kg = ?, pose_profile = ?, cues = ?, camera_hint = ?, updated_at = ?
		 WHERE id = ?`,
		e.Name, e.Description, e.Category, boolToInt(e.Unilateral), e.DefaultWeightKg, e.PoseProfile, e.Cues, e.CameraHint, Now(), e.ID)
	if err != nil {
		return e, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return e, ErrNotFound
	}
	return s.GetExercise(ctx, e.ID)
}

func (s *Store) DeleteExercise(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM exercises WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
