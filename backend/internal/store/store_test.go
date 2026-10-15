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
	if want := len(seedExercises) + len(seedExercisesV2) + len(seedExercisesV3); len(ex) != want {
		t.Fatalf("got %d exercises, want %d", len(ex), want)
	}
	wo, _ := s.ListWorkouts(ctx)
	if len(wo) != 6 {
		t.Fatalf("got %d workouts, want 6", len(wo))
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

// Eine DB aus der Zeit vor der Seed-Versionierung bekommt beim Update die v2-Daten,
// ohne dass Übungen doppelt angelegt werden; vom Nutzer gelöschte v1-Workouts bleiben weg.
func TestSeedUpgradeFromV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	exBefore, _ := s.ListExercises(ctx)
	// Zustand „altes Image“: nur v1-Seed, kein seedVersion, v2-Workouts fehlen.
	for _, q := range []string{
		`DELETE FROM settings WHERE key = 'seedVersion'`,
		`DELETE FROM workouts WHERE name LIKE 'Workout A%' OR name LIKE 'Workout B%' OR name = 'Tabata Swings'`,
		`DELETE FROM exercises WHERE name = 'Dead Bug'`,
	} {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	ex, _ := s2.ListExercises(ctx)
	if len(ex) != len(exBefore) {
		t.Fatalf("exercises: got %d, want %d (no duplicates, Dead Bug re-created)", len(ex), len(exBefore))
	}
	wo, _ := s2.ListWorkouts(ctx)
	names := map[string]bool{}
	for _, w := range wo {
		names[w.Name] = true
	}
	if !names["Workout A – Hüftstreckung & Stabilität"] || !names["Workout B – Einbeinig & Hüfte seitlich"] {
		t.Fatalf("v2 workouts missing: %v", names)
	}
	if names["Tabata Swings"] {
		t.Fatal("deleted v1 workout must not come back")
	}
	m, _ := s2.GetSettings(ctx)
	if string(m["seedVersion"]) != "3" {
		t.Fatalf("seedVersion %s", m["seedVersion"])
	}
}

// v2→v3: unveränderte A/B-Workouts bekommen Aufwärmen + neue Pausen,
// im Editor angepasste bleiben unangetastet.
func TestSeedUpgradeFromV2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ids := map[string]int64{}
	ex, _ := s.ListExercises(ctx)
	for _, e := range ex {
		ids[e.Name] = e.ID
	}
	// Datenbank auf v2-Stand zurücksetzen: A/B im v2-Format, B zusätzlich vom Nutzer geändert.
	for _, w := range seedWorkoutsV2(ids) {
		raw, _ := json.Marshal(w.structure)
		notes := w.notes
		if w.name[len("Workout ")] == 'B' {
			notes = "meine eigene Notiz"
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE workouts SET notes = ?, structure = ? WHERE name = ?`, notes, string(raw), w.name); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.PutSettings(ctx, map[string]json.RawMessage{"seedVersion": json.RawMessage("2")}); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	wo, _ := s2.ListWorkouts(ctx)
	blocks := func(w Workout) []map[string]any {
		var st struct {
			Blocks []map[string]any `json:"blocks"`
		}
		json.Unmarshal(w.Structure, &st)
		return st.Blocks
	}
	var a, b *Workout
	for i := range wo {
		switch wo[i].Name {
		case "Workout A – Hüftstreckung & Stabilität":
			a = &wo[i]
		case "Workout B – Einbeinig & Hüfte seitlich":
			b = &wo[i]
		}
	}
	if a == nil || b == nil {
		t.Fatal("A/B missing")
	}
	if bl := blocks(*a); len(bl) != 7 || bl[0]["type"] != "circuit" {
		t.Fatalf("A not upgraded: %d blocks", len(bl))
	}
	if bl := blocks(*b); len(bl) != 7 || b.Notes != "meine eigene Notiz" {
		t.Fatalf("modified B must stay untouched: %d blocks, notes %q", len(bl), b.Notes)
	}
}

// Alle Übungs-IDs in den Seed-Workouts müssen existieren.
func TestSeedWorkoutsReferenceExistingExercises(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	ex, _ := s.ListExercises(ctx)
	valid := map[float64]bool{}
	for _, e := range ex {
		valid[float64(e.ID)] = true
	}
	wo, _ := s.ListWorkouts(ctx)
	for _, w := range wo {
		var st struct {
			Blocks []struct {
				Item *struct {
					ExerciseID float64 `json:"exerciseId"`
				} `json:"item"`
				Items []struct {
					ExerciseID float64 `json:"exerciseId"`
				} `json:"items"`
			} `json:"blocks"`
		}
		if err := json.Unmarshal(w.Structure, &st); err != nil {
			t.Fatal(err)
		}
		for _, b := range st.Blocks {
			if b.Item != nil && !valid[b.Item.ExerciseID] {
				t.Errorf("%s: unknown exercise %v", w.Name, b.Item.ExerciseID)
			}
			for _, it := range b.Items {
				if !valid[it.ExerciseID] {
					t.Errorf("%s: unknown exercise %v", w.Name, it.ExerciseID)
				}
			}
		}
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
