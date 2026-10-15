package store

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

// Seed v3: vollständiger Plan für Workout A/B – Aufwärmen, Pausen 60–90 s, Trainingsempfehlung.
// Bereits vorhandene A/B-Workouts werden nur aktualisiert, wenn sie noch exakt dem
// v2-Stand entsprechen (also nicht im Editor verändert wurden).

var seedExercisesV3 = []seedExercise{
	{"Glute Bridge", "core", false, 0, "", "seitlich",
		"Hüftheben in Rückenlage auf beiden Beinen.",
		"Fersen in den Boden drücken\nOben Po anspannen, kein Hohlkreuz\nKurz halten, kontrolliert absenken"},
}

const planNotes = "Plan: 2× pro Woche, je ca. 40 Min., mind. ein Tag Pause dazwischen (z. B. Mo + Do), " +
	"im Wechsel mit Workout %s. Im Herbst/Winter sind auch 3× möglich (A–B–A, nächste Woche B–A–B). " +
	"Nicht auf den Tag vor einer harten oder langen Ausfahrt legen.\n" +
	"Fokus gegen Knie-/Hüftbeschwerden: Gesäß (v. a. seitlicher Gesäßmuskel), Beinrückseite, " +
	"einbeinige Stabilität und Rumpf.\n" +
	"Pausen 60–90 s zwischen den Sätzen; einseitige Übungen links/rechts direkt im Wechsel."

func (s *Store) seedV3(ctx context.Context) error {
	needed := map[string]bool{
		"Two-Hand Swing": true, "Goblet Squat": true, "Single-Leg Deadlift": true, "Reverse Lunge": true,
		"Suitcase Carry": true, "Halo": true,
	}
	list := append(append([]seedExercise{}, seedExercisesV2...), seedExercisesV3...)
	for _, se := range seedExercises {
		if needed[se.Name] {
			list = append(list, se)
		}
	}
	ids, err := s.ensureExercises(ctx, list)
	if err != nil {
		return err
	}

	previous := map[string]seedWorkout{}
	for _, w := range seedWorkoutsV2(ids) {
		previous[w.name] = w
	}
	existing, err := s.ListWorkouts(ctx)
	if err != nil {
		return err
	}
	byName := map[string]Workout{}
	for _, w := range existing {
		byName[w.Name] = w
	}

	for _, w := range seedWorkoutsV3(ids) {
		raw, err := json.Marshal(w.structure)
		if err != nil {
			return err
		}
		cur, ok := byName[w.name]
		if !ok {
			if _, err := s.CreateWorkout(ctx, Workout{Name: w.name, Notes: w.notes, Structure: raw}); err != nil {
				return err
			}
			continue
		}
		prev, known := previous[w.name]
		if !known || cur.Notes != prev.notes || !sameJSON(cur.Structure, prev.structure) {
			continue // vom Nutzer angepasst → nicht überschreiben
		}
		cur.Notes = w.notes
		cur.Structure = raw
		if _, err := s.UpdateWorkout(ctx, cur); err != nil {
			return err
		}
	}
	return nil
}

// sameJSON vergleicht gespeichertes JSON mit einer Go-Struktur inhaltlich.
func sameJSON(stored json.RawMessage, v any) bool {
	want, err := json.Marshal(v)
	if err != nil {
		return false
	}
	var a, b any
	if json.Unmarshal(stored, &a) != nil || json.Unmarshal(want, &b) != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func seedWorkoutsV3(ids map[string]int64) []seedWorkout {
	blockID := 0
	nextID := func() string { blockID++; return fmt.Sprintf("seed3-%d", blockID) }
	sets := func(title string, sets, rest int, it map[string]any, record bool) map[string]any {
		b := map[string]any{"id": nextID(), "type": "sets", "title": title, "sets": sets, "restSec": rest, "item": it}
		if record {
			b["record"] = true
		}
		return b
	}
	warmup := func() map[string]any {
		return map[string]any{
			"id": nextID(), "type": "circuit", "title": "Aufwärmen · Squats langsam, unten halten", "rounds": 1,
			"restBetweenItemsSec": 0, "restBetweenRoundsSec": 0,
			"items": []any{
				item(ids["Glute Bridge"], "bw", 10, 0),
				item(ids["Halo"], 12, 10, 0),
				item(ids["Goblet Squat"], 12, 5, 0),
				// „10 Ausfallschritte“ = 5 je Seite
				item(ids["Reverse Lunge"], "bw", 5, 0),
			},
		}
	}
	structure := func(blocks ...map[string]any) map[string]any {
		return map[string]any{"version": 1, "prepareSec": 10, "blockRestSec": 60, "blocks": blocks}
	}
	return []seedWorkout{
		{
			name:  "Workout A – Hüftstreckung & Stabilität",
			notes: fmt.Sprintf(planNotes, "B"),
			structure: structure(
				warmup(),
				sets("Kettlebell Swing", 5, 60, item(ids["Two-Hand Swing"], 16, 15, 0), true),
				sets("Goblet Squat · 3 Sek. runter", 3, 90, item(ids["Goblet Squat"], 16, 10, 0), true),
				sets("Einbeiniges Kreuzheben", 3, 60, item(ids["Single-Leg Deadlift"], 12, 8, 0), false),
				sets("Reverse Lunge · Goblet", 3, 60, item(ids["Reverse Lunge"], 12, 8, 0), false),
				sets("Suitcase Carry", 3, 60, distItem(ids["Suitcase Carry"], 16, 30), false),
				sets("Seitstütz", 3, 60, item(ids["Seitstütz"], "bw", 0, 30), false),
			),
		},
		{
			name:  "Workout B – Einbeinig & Hüfte seitlich",
			notes: fmt.Sprintf(planNotes, "A") + "\nKreuzheben mit 16 + 12 kg (= 28 kg gesamt).",
			structure: structure(
				warmup(),
				sets("Kreuzheben · 16 + 12 kg", 4, 90, item(ids["Kreuzheben (2 Kettlebells)"], 28, 8, 0), true),
				sets("Bulgarian Split Squat", 3, 60, item(ids["Bulgarian Split Squat"], 12, 8, 0), false),
				sets("Step-ups · Stuhl/Bank", 3, 60, item(ids["Step-up"], 12, 8, 0), false),
				sets("Seitlicher Ausfallschritt", 3, 60, item(ids["Seitlicher Ausfallschritt"], 12, 6, 0), false),
				sets("Einbeinige Glute Bridge · 12 kg auf der Hüfte", 3, 60, item(ids["Einbeinige Glute Bridge"], 12, 10, 0), false),
				sets("Copenhagen Plank · Knie aufgelegt", 3, 60, item(ids["Copenhagen Plank"], "bw", 0, 20), false),
				sets("Dead Bug", 3, 60, item(ids["Dead Bug"], "bw", 8, 0), false),
			),
		},
	}
}
