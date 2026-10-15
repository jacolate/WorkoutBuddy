package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"workoutapp/internal/media"
	"workoutapp/internal/store"
)

func (s *Server) listRecordings(w http.ResponseWriter, r *http.Request) {
	var sessionID *int64
	if v := r.URL.Query().Get("sessionId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid sessionId")
			return
		}
		sessionID = &id
	}
	list, err := s.store.ListRecordings(r.Context(), sessionID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getRecording(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	rec, err := s.store.GetRecording(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func optionalID(v string) (*int64, error) {
	if v == "" || v == "null" {
		return nil, nil
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// uploadRecording nimmt ein multipart-Formular entgegen und streamt die Datei direkt auf die Platte.
// Felder: file (Pflicht), sessionId, exerciseId, durationSec, label, note.
func (s *Server) uploadRecording(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUploadBytes)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "multipart/form-data erwartet")
		return
	}

	var rec store.Recording
	fields := map[string]string{}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.cleanupUpload(rec.Filename)
			uploadError(w, err)
			return
		}
		if part.FormName() == "file" {
			if rec.Filename != "" {
				s.cleanupUpload(rec.Filename)
				writeError(w, http.StatusBadRequest, "nur eine Datei erlaubt")
				return
			}
			mime := part.Header.Get("Content-Type")
			if !media.IsAllowedMime(mime) {
				writeError(w, http.StatusUnsupportedMediaType, "nur Videodateien (mp4/webm/mov)")
				return
			}
			name, size, err := s.media.Save(part, mime)
			if err != nil {
				uploadError(w, err)
				return
			}
			rec.Filename, rec.SizeBytes, rec.Mime = name, size, strings.TrimSpace(strings.SplitN(mime, ";", 2)[0])
			continue
		}
		b, err := io.ReadAll(io.LimitReader(part, 4096))
		if err != nil {
			s.cleanupUpload(rec.Filename)
			uploadError(w, err)
			return
		}
		fields[part.FormName()] = string(b)
	}
	if rec.Filename == "" {
		writeError(w, http.StatusBadRequest, "Datei fehlt")
		return
	}

	bad := func(msg string) {
		s.cleanupUpload(rec.Filename)
		writeError(w, http.StatusBadRequest, msg)
	}
	if rec.SessionID, err = optionalID(fields["sessionId"]); err != nil {
		bad("invalid sessionId")
		return
	}
	if rec.ExerciseID, err = optionalID(fields["exerciseId"]); err != nil {
		bad("invalid exerciseId")
		return
	}
	if rec.SessionID != nil {
		if _, err := s.store.GetSession(r.Context(), *rec.SessionID); err != nil {
			rec.SessionID = nil
		}
	}
	if rec.ExerciseID != nil {
		if _, err := s.store.GetExercise(r.Context(), *rec.ExerciseID); err != nil {
			rec.ExerciseID = nil
		}
	}
	if v := fields["durationSec"]; v != "" {
		rec.DurationSec, _ = strconv.ParseFloat(v, 64)
	}
	rec.Label = fields["label"]
	rec.Note = fields["note"]

	created, err := s.store.CreateRecording(r.Context(), rec)
	if err != nil {
		s.cleanupUpload(rec.Filename)
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) cleanupUpload(name string) {
	if name != "" {
		s.media.Delete(name)
	}
}

func uploadError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "Datei zu groß")
		return
	}
	writeError(w, http.StatusBadRequest, "Upload fehlgeschlagen: "+err.Error())
}

func (s *Server) patchRecording(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	rec, err := s.store.UpdateRecordingNote(r.Context(), id, body.Note)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) deleteRecording(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	name, err := s.store.DeleteRecording(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	if err := s.media.Delete(name); err != nil {
		handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// recordingFile streamt die Videodatei mit Range-Support (nötig für Safari und Spulen).
func (s *Server) recordingFile(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	rec, err := s.store.GetRecording(r.Context(), id)
	if err != nil {
		handleErr(w, err)
		return
	}
	p, err := s.media.Path(rec.Filename)
	if err != nil {
		handleErr(w, err)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		handleErr(w, store.ErrNotFound)
		return
	}
	defer f.Close()
	modTime := time.Time{}
	if st, err := f.Stat(); err == nil {
		modTime = st.ModTime()
	}
	// middleware.NoCache auf /api verhindert Caching; Videos ändern sich nie, also privat cachen.
	w.Header().Del("Pragma")
	w.Header().Del("Expires")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Content-Type", rec.Mime)
	http.ServeContent(w, r, rec.Filename, modTime, f)
}
