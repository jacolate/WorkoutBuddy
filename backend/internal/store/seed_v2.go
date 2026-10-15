package store

import (
	"context"
	"fmt"
)

// Seed v2 (eingefroren – dient in v3 als Vergleichsstand): Übungen und Workouts „A – Hüftstreckung & Stabilität“ und „B – Einbeinig & Hüfte seitlich“.

var seedExercisesV2 = []seedExercise{
	{"Kreuzheben (2 Kettlebells)", "hinge", false, 28, "hinge", "seitlich",
		"Kreuzheben mit je einer Kettlebell pro Hand (z. B. 16 + 12 kg), Bells neben den Füßen.",
		"Hüfte nach hinten, Rücken neutral\nBeide Bells nah am Körper\nOben ganz aufrichten, Po anspannen\nBei ungleichen Gewichten nicht zur schweren Seite kippen"},
	{"Bulgarian Split Squat", "lunge", true, 12, "", "seitlich",
		"Ausfallschritt mit dem hinteren Fuß erhöht auf Stuhl oder Bank, Kettlebell im Goblet-Griff.",
		"Vorderes Knie zeigt Richtung Zehen\nOberkörper leicht nach vorn, Gewicht auf dem vorderen Bein\nKontrolliert absenken, hinteres Knie Richtung Boden"},
	{"Step-up", "lunge", true, 12, "", "seitlich",
		"Aufsteigen auf Stuhl oder Bank, Kettlebell im Goblet-Griff oder seitlich.",
		"Ganzer Fuß auf der Erhöhung\nAus dem oberen Bein drücken, nicht vom unteren abstoßen\nLangsam wieder absteigen"},
	{"Seitlicher Ausfallschritt", "lunge", true, 12, "", "frontal",
		"Ausfallschritt zur Seite, Kettlebell im Goblet-Griff.",
		"Hüfte nach hinten schieben, Standbein bleibt gestreckt\nKnie über dem Fuß\nFuß bleibt ganz am Boden"},
	{"Einbeinige Glute Bridge", "core", true, 12, "", "seitlich",
		"Hüftheben in Rückenlage auf einem Bein, Kettlebell auf der Hüfte.",
		"Ferse in den Boden drücken\nBecken bleibt waagerecht, nicht zur Seite kippen\nOben kurz halten, Po anspannen"},
	{"Seitstütz", "core", true, 0, "", "frontal",
		"Seitlicher Unterarmstütz.",
		"Ellbogen unter der Schulter\nKörper bildet eine Linie, Hüfte nicht absinken lassen"},
	{"Copenhagen Plank", "core", true, 0, "", "frontal",
		"Seitstütz mit dem oberen Bein (Knie) auf einer Bank aufgelegt – Adduktoren.",
		"Knie aufgelegt (leichtere Variante)\nHüfte oben halten, Körper in einer Linie\nUnteres Bein darf locker hängen"},
	{"Dead Bug", "core", true, 0, "", "seitlich",
		"Rückenlage, gegengleich Arm und Bein strecken.",
		"Unterer Rücken bleibt am Boden\nLangsam und kontrolliert ausatmen\nGegenüberliegenden Arm und Bein gleichzeitig strecken"},
}

func (s *Store) seedV2(ctx context.Context) error {
	// Aus v1 nur die Übungen sicherstellen, die die neuen Workouts brauchen – andere,
	// vom Nutzer gelöschte Übungen sollen nicht zurückkommen.
	needed := map[string]bool{"Two-Hand Swing": true, "Goblet Squat": true, "Single-Leg Deadlift": true, "Reverse Lunge": true, "Suitcase Carry": true}
	list := append([]seedExercise{}, seedExercisesV2...)
	for _, se := range seedExercises {
		if needed[se.Name] {
			list = append(list, se)
		}
	}
	ids, err := s.ensureExercises(ctx, list)
	if err != nil {
		return err
	}
	return s.ensureWorkouts(ctx, seedWorkoutsV2(ids))
}

func distItem(id int64, weight any, meters int) map[string]any {
	return map[string]any{"exerciseId": id, "weight": weight, "distanceM": meters}
}

func seedWorkoutsV2(ids map[string]int64) []seedWorkout {
	blockID := 0
	nextID := func() string { blockID++; return fmt.Sprintf("seed2-%d", blockID) }
	sets := func(title string, sets, rest int, it map[string]any, record bool) map[string]any {
		b := map[string]any{"id": nextID(), "type": "sets", "title": title, "sets": sets, "restSec": rest, "item": it}
		if record {
			b["record"] = true
		}
		return b
	}
	structure := func(blocks ...map[string]any) map[string]any {
		return map[string]any{"version": 1, "prepareSec": 10, "blockRestSec": 60, "blocks": blocks}
	}
	return []seedWorkout{
		{
			name:  "Workout A – Hüftstreckung & Stabilität",
			notes: "Pausen sind Vorschläge – bei Bedarf im Editor anpassen. Einseitige Übungen laufen automatisch links und rechts.",
			structure: structure(
				sets("Kettlebell Swing", 5, 60, item(ids["Two-Hand Swing"], 16, 15, 0), true),
				sets("Goblet Squat · 3 Sek. runter", 3, 90, item(ids["Goblet Squat"], 16, 10, 0), true),
				sets("Einbeiniges Kreuzheben", 3, 60, item(ids["Single-Leg Deadlift"], 12, 8, 0), false),
				sets("Reverse Lunge · Goblet", 3, 60, item(ids["Reverse Lunge"], 12, 8, 0), false),
				sets("Suitcase Carry", 3, 60, distItem(ids["Suitcase Carry"], 16, 30), false),
				sets("Seitstütz", 3, 30, item(ids["Seitstütz"], "bw", 0, 30), false),
			),
		},
		{
			name:  "Workout B – Einbeinig & Hüfte seitlich",
			notes: "Kreuzheben mit 16 + 12 kg (= 28 kg gesamt). Pausen sind Vorschläge – bei Bedarf im Editor anpassen.",
			structure: structure(
				sets("Kreuzheben · 16 + 12 kg", 4, 90, item(ids["Kreuzheben (2 Kettlebells)"], 28, 8, 0), true),
				sets("Bulgarian Split Squat", 3, 60, item(ids["Bulgarian Split Squat"], 12, 8, 0), false),
				sets("Step-ups · Stuhl/Bank", 3, 60, item(ids["Step-up"], 12, 8, 0), false),
				sets("Seitlicher Ausfallschritt", 3, 60, item(ids["Seitlicher Ausfallschritt"], 12, 6, 0), false),
				sets("Einbeinige Glute Bridge · 12 kg auf der Hüfte", 3, 45, item(ids["Einbeinige Glute Bridge"], 12, 10, 0), false),
				sets("Copenhagen Plank · Knie aufgelegt", 3, 30, item(ids["Copenhagen Plank"], "bw", 0, 20), false),
				sets("Dead Bug", 3, 30, item(ids["Dead Bug"], "bw", 8, 0), false),
			),
		},
	}
}
