package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSeedAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ex, _ := s.ListExercises(ctx)
	if len(ex) != len(seedExercises) {
		t.Fatalf("got %d exercises, want %d", len(ex), len(seedExercises))
	}
	wo, _ := s.ListWorkouts(ctx)
	if len(wo) != 4 {
		t.Fatalf("got %d workouts, want 4", len(wo))
	}
	for _, w := range wo {
		var v map[string]any
		if err := json.Unmarshal(w.Structure, &v); err != nil {
			t.Fatalf("workout %q: invalid structure: %v", w.Name, err)
		}
	}
	s.Close()

	// Zweites Öffnen darf weder migrieren noch erneut seeden.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	ex2, _ := s2.ListExercises(ctx)
	if len(ex2) != len(ex) {
		t.Fatalf("reseeded: %d vs %d", len(ex2), len(ex))
	}
}

func TestExerciseCRUD(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	w := 16.0
	p := "squat"
	e, err := s.CreateExercise(ctx, Exercise{Name: "Test Squat", Category: "squat", DefaultWeightKg: &w, PoseProfile: &p, Unilateral: true})
	if err != nil {
		t.Fatal(err)
	}
	if !e.Unilateral || *e.DefaultWeightKg != 16 || *e.PoseProfile != "squat" {
		t.Fatalf("unexpected %+v", e)
	}
	e.Name = "Renamed"
	e.PoseProfile = nil
	e2, err := s.UpdateExercise(ctx, e)
	if err != nil {
		t.Fatal(err)
	}
	if e2.Name != "Renamed" || e2.PoseProfile != nil {
		t.Fatalf("update failed: %+v", e2)
	}
	if err := s.DeleteExercise(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetExercise(ctx, e.ID); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSessionAndRecordings(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	wo, _ := s.ListWorkouts(ctx)
	wid := wo[0].ID
	se, err := s.CreateSession(ctx, Session{WorkoutID: &wid, WorkoutName: wo[0].Name, Snapshot: wo[0].Structure})
	if err != nil {
		t.Fatal(err)
	}
	if string(se.Log) != "[]" {
		t.Fatalf("log default: %s", se.Log)
	}
	fin := Now()
	log := json.RawMessage(`[{"reps":10}]`)
	se, err = s.PatchSession(ctx, se.ID, SessionPatch{FinishedAt: &fin, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	if se.FinishedAt == nil || string(se.Log) != `[{"reps":10}]` {
		t.Fatalf("patch failed: %+v", se)
	}

	ex, _ := s.ListExercises(ctx)
	rec, err := s.CreateRecording(ctx, Recording{SessionID: &se.ID, ExerciseID: &ex[0].ID, Filename: "abc.webm", Mime: "video/webm"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.ExerciseName == nil || *rec.ExerciseName != ex[0].Name {
		t.Fatalf("join failed: %+v", rec)
	}
	got, _ := s.GetSession(ctx, se.ID)
	if got.RecordingCount != 1 {
		t.Fatalf("recording count %d", got.RecordingCount)
	}
	// Workout löschen setzt workout_id auf NULL, Session bleibt.
	if err := s.DeleteWorkout(ctx, wid); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetSession(ctx, se.ID)
	if got.WorkoutID != nil {
		t.Fatalf("workout id not nulled")
	}
	list, _ := s.ListWorkouts(ctx)
	for _, w := range list {
		if w.ID == wid {
			t.Fatal("workout not deleted")
		}
	}
}

func TestAuthSessions(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	if err := s.CreateAuthSession(ctx, "valid", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAuthSession(ctx, "expired", time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthSessionExpiry(ctx, "valid"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthSessionExpiry(ctx, "expired"); err != ErrNotFound {
		t.Fatalf("expired session accepted: %v", err)
	}
	if err := s.PurgeExpiredAuthSessions(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsMerge(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	if err := s.PutSettings(ctx, map[string]json.RawMessage{"voice": json.RawMessage(`false`)}); err != nil {
		t.Fatal(err)
	}
	m, _ := s.GetSettings(ctx)
	if string(m["voice"]) != "false" || string(m["weights"]) != "[12, 16]" {
		t.Fatalf("unexpected settings %v", m)
	}
}
