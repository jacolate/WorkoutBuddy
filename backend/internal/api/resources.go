package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"workoutapp/internal/store"
)

var validPoseProfiles = map[string]bool{"swing": true, "squat": true, "hinge": true, "press": true}

// --- Übungen ---

func validateExercise(e *store.Exercise) string {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		return "Name fehlt"
	}
	if e.Category == "" {
		e.Category = "other"
	}
	if e.PoseProfile != nil {
		if *e.PoseProfile == "" {
			e.PoseProfile = nil
		} else if !validPoseProfiles[*e.PoseProfile] {
			return "unbekanntes Pose-Profil"
		}
	}
	if e.DefaultWeightKg != nil && (*e.DefaultWeightKg < 0 || *e.DefaultWeightKg > 200) {
		return "ungültiges Gewicht"
	}
	return ""
}

func (s *Server) listExercises(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListExercises(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	e, err := s.store.GetExercise(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) createExercise(w http.ResponseWriter, r *http.Request) {
	var e store.Exercise
	if !decodeJSON(w, r, &e) {
		return
	}
	if msg := validateExercise(&e); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	created, err := s.store.CreateExercise(r.Context(), e)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var e store.Exercise
	if !decodeJSON(w, r, &e) {
		return
	}
	e.ID = id
	if msg := validateExercise(&e); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	updated, err := s.store.UpdateExercise(r.Context(), e)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteExercise(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteExercise(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Workouts ---

func validateWorkout(wo *store.Workout) string {
	wo.Name = strings.TrimSpace(wo.Name)
	if wo.Name == "" {
		return "Name fehlt"
	}
	var shape struct {
		Blocks []json.RawMessage `json:"blocks"`
	}
	if len(wo.Structure) == 0 || !bytes.HasPrefix(bytes.TrimSpace(wo.Structure), []byte("{")) {
		return "structure muss ein Objekt sein"
	}
	if err := json.Unmarshal(wo.Structure, &shape); err != nil || shape.Blocks == nil {
		return "structure.blocks fehlt"
	}
	return ""
}

func (s *Server) listWorkouts(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListWorkouts(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	wo, err := s.store.GetWorkout(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wo)
}

func (s *Server) createWorkout(w http.ResponseWriter, r *http.Request) {
	var wo store.Workout
	if !decodeJSON(w, r, &wo) {
		return
	}
	if msg := validateWorkout(&wo); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	created, err := s.store.CreateWorkout(r.Context(), wo)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var wo store.Workout
	if !decodeJSON(w, r, &wo) {
		return
	}
	wo.ID = id
	if msg := validateWorkout(&wo); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	updated, err := s.store.UpdateWorkout(r.Context(), wo)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteWorkout(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) duplicateWorkout(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	wo, err := s.store.GetWorkout(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	wo.Name += " (Kopie)"
	created, err := s.store.CreateWorkout(r.Context(), wo)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// --- Sessions ---

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListSessions(r.Context(), 200)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	se, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, se)
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var se store.Session
	if !decodeJSON(w, r, &se) {
		return
	}
	if len(se.Snapshot) == 0 {
		writeError(w, http.StatusBadRequest, "snapshot fehlt")
		return
	}
	if se.WorkoutID != nil {
		if _, err := s.store.GetWorkout(r.Context(), *se.WorkoutID); err != nil {
			se.WorkoutID = nil
		}
	}
	created, err := s.store.CreateSession(r.Context(), se)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) patchSession(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var p store.SessionPatch
	if !decodeJSON(w, r, &p) {
		return
	}
	se, err := s.store.PatchSession(r.Context(), id, p)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, se)
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	// Aufnahmen der Session mitlöschen (inkl. Dateien).
	recs, err := s.store.ListRecordings(r.Context(), &id)
	if err != nil {
		handleErr(w, err)
		return
	}
	for _, rec := range recs {
		if name, err := s.store.DeleteRecording(r.Context(), rec.ID); err == nil {
			s.media.Delete(name)
		}
	}
	if err := s.store.DeleteSession(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Einstellungen ---

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	m, err := s.store.GetSettings(r.Context())
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	var m map[string]json.RawMessage
	if !decodeJSON(w, r, &m) {
		return
	}
	delete(m, "seeded")
	if err := s.store.PutSettings(r.Context(), m); err != nil {
		handleErr(w, err)
		return
	}
	s.getSettings(w, r)
}
