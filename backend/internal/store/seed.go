package store

import (
	"context"
	"encoding/json"
	"fmt"
)

type seedExercise struct {
	Name        string
	Category    string
	Unilateral  bool
	Weight      float64 // 0 = Körpergewicht/ohne
	Pose        string
	CameraHint  string
	Description string
	Cues        string
}

var seedExercises = []seedExercise{
	{"Two-Hand Swing", "swing", false, 16, "swing", "seitlich",
		"Explosive Hüftstreckung, die Kettlebell schwingt bis etwa Brusthöhe.",
		"Hüfte nach hinten schieben, nicht in die Knie gehen\nOben Po und Bauch anspannen, Körper gerade\nArme sind nur Seile – die Kraft kommt aus der Hüfte\nKettlebell hoch zwischen die Beine (Hike Pass)"},
	{"One-Arm Swing", "swing", true, 12, "swing", "seitlich",
		"Swing mit einer Hand, zusätzliche Anti-Rotations-Arbeit für den Rumpf.",
		"Schultern bleiben parallel zum Boden\nLat aktiv, Schulter weg vom Ohr\nVolle Hüftstreckung oben"},
	{"Goblet Squat", "squat", false, 16, "squat", "seitlich",
		"Kniebeuge mit der Kettlebell vor der Brust an den Hörnern gehalten.",
		"Ellbogen zwischen die Knie\nBrust aufrecht, Blick nach vorn\nFersen bleiben am Boden\nMindestens Hüfte auf Kniehöhe"},
	{"Front Squat (Rack)", "squat", true, 12, "squat", "seitlich",
		"Kniebeuge mit der Kettlebell in Rack-Position.",
		"Unterarm vertikal, Kettlebell am Körper\nRumpf fest, nicht zur Seite kippen\nKnie zeigen in Richtung Zehen"},
	{"Kettlebell Deadlift", "hinge", false, 16, "hinge", "seitlich",
		"Kreuzheben mit der Kettlebell zwischen den Füßen.",
		"Rücken neutral, Hüfte nach hinten\nSchienbeine fast senkrecht\nOben vollständig aufrichten, nicht überstrecken"},
	{"Single-Leg Deadlift", "hinge", true, 12, "", "seitlich",
		"Einbeiniges Kreuzheben für Balance und hintere Kette.",
		"Hüfte bleibt parallel zum Boden\nStandbein leicht gebeugt\nHinteres Bein und Oberkörper bilden eine Linie"},
	{"Clean", "clean", true, 12, "", "frontal",
		"Kettlebell aus dem Schwung in die Rack-Position bringen.",
		"Kettlebell nah am Körper führen (Reißverschluss)\nHand um die Kugel „einfädeln“, nicht auf den Unterarm knallen lassen\nOben stabil im Rack ankommen"},
	{"Press", "press", true, 12, "press", "frontal",
		"Strikter Überkopfdruck aus der Rack-Position.",
		"Gesäß und Bauch fest, kein Hohlkreuz\nUnterarm bleibt senkrecht\nOben Arm ganz strecken, Bizeps neben dem Ohr"},
	{"Push Press", "press", true, 16, "press", "frontal",
		"Überkopfdruck mit Schwung aus einem kurzen Dip der Beine.",
		"Kurzer, aufrechter Dip – nicht nach vorne kippen\nBeine treiben, Arm vollendet\nOben kurz halten"},
	{"Snatch", "swing", true, 12, "", "seitlich",
		"Kettlebell in einer Bewegung vom Schwung über den Kopf.",
		"Kraft aus der Hüfte wie beim Swing\nKettlebell eng am Körper, oben einfädeln\nOben Arm gestreckt und stabil"},
	{"Turkish Get-up", "core", true, 12, "", "seitlich",
		"Vom Liegen zum Stand und zurück, Kettlebell immer über Kopf.",
		"Blick zur Kettlebell\nArm bleibt senkrecht und gestreckt\nLangsam und kontrolliert"},
	{"Bent-over Row", "pull", true, 16, "", "seitlich",
		"Einarmiges Rudern im Hinge-Stand.",
		"Rücken gerade, Hüfte gebeugt\nEllbogen Richtung Hüfte ziehen\nSchulterblatt aktiv nach hinten"},
	{"Halo", "mobility", false, 12, "", "frontal",
		"Kettlebell kreist um den Kopf – Schultermobilität und Warm-up.",
		"Rumpf fest, kein Hohlkreuz\nKettlebell nah am Kopf führen\nRichtung wechseln"},
	{"Reverse Lunge", "lunge", true, 12, "", "seitlich",
		"Ausfallschritt nach hinten mit Kettlebell (Goblet oder Rack).",
		"Vorderes Knie über dem Fuß\nHinteres Knie fast zum Boden\nOberkörper aufrecht"},
	{"Windmill", "core", true, 12, "", "frontal",
		"Seitliche Hüftbeuge mit Kettlebell über Kopf.",
		"Blick zur Kettlebell\nHüfte zur Seite der Kettlebell schieben\nArm oben bleibt senkrecht"},
	{"Thruster", "squat", false, 12, "squat", "seitlich",
		"Goblet- oder Front-Squat direkt in einen Überkopfdruck.",
		"Aus der Tiefe explosiv hochdrücken\nSchwung aus den Beinen in den Druck übertragen"},
	{"High Pull", "pull", true, 12, "", "seitlich",
		"Swing, bei dem die Kettlebell oben zum Körper gezogen wird.",
		"Hüftstreckung zuerst, dann ziehen\nEllbogen hoch und nach hinten"},
	{"Figure 8", "core", false, 12, "", "frontal",
		"Kettlebell in Achterform zwischen den Beinen durchreichen.",
		"Hinge-Position halten\nRücken neutral"},
	{"Suitcase Carry", "carry", true, 16, "", "",
		"Gehen mit einer Kettlebell auf einer Seite.",
		"Nicht zur Seite kippen\nSchultern gerade, Rumpf fest"},
	{"Farmer Carry", "carry", false, 16, "", "",
		"Gehen mit zwei Kettlebells (12 + 16 kg geht auch).",
		"Aufrecht gehen, Schultern zurück\nKurze, kontrollierte Schritte"},
	{"Push-up", "bodyweight", false, 0, "", "seitlich",
		"Liegestütz.",
		"Körper bildet eine Linie\nEllbogen ca. 45° zum Körper"},
	{"Plank", "core", false, 0, "", "seitlich",
		"Unterarmstütz.",
		"Po anspannen, Becken leicht kippen\nNicht durchhängen"},
	{"Burpee", "bodyweight", false, 0, "", "",
		"Liegestütz, aufspringen, Strecksprung.",
		"Kontrolliert landen\nRumpf fest in der Plank-Phase"},
	{"Mountain Climber", "bodyweight", false, 0, "", "",
		"Im Stütz abwechselnd Knie zur Brust ziehen.",
		"Hüfte tief halten\nSchultern über den Händen"},
}

// seed füllt eine frische Datenbank mit Übungen, Beispiel-Workouts und Einstellungen.
func (s *Store) seed(ctx context.Context) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM settings WHERE key = 'seeded'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	ids := map[string]int64{}
	for _, se := range seedExercises {
		e := Exercise{
			Name: se.Name, Description: se.Description, Category: se.Category, Unilateral: se.Unilateral,
			Cues: se.Cues, CameraHint: se.CameraHint,
		}
		if se.Weight > 0 {
			w := se.Weight
			e.DefaultWeightKg = &w
		}
		if se.Pose != "" {
			p := se.Pose
			e.PoseProfile = &p
		}
		created, err := s.CreateExercise(ctx, e)
		if err != nil {
			return err
		}
		ids[se.Name] = created.ID
	}

	for _, w := range seedWorkouts(ids) {
		raw, err := json.Marshal(w.structure)
		if err != nil {
			return err
		}
		if _, err := s.CreateWorkout(ctx, Workout{Name: w.name, Notes: w.notes, Structure: raw}); err != nil {
			return err
		}
	}

	return s.PutSettings(ctx, map[string]json.RawMessage{
		"seeded":          json.RawMessage(`true`),
		"weights":         json.RawMessage(`[12, 16]`),
		"sound":           json.RawMessage(`true`),
		"voice":           json.RawMessage(`true`),
		"prepareSec":      json.RawMessage(`10`),
		"secondsPerRep":   json.RawMessage(`3`),
		"autoAdvanceReps": json.RawMessage(`false`),
	})
}

type seedWorkout struct {
	name, notes string
	structure   map[string]any
}

func item(id int64, weight any, reps, dur int) map[string]any {
	m := map[string]any{"exerciseId": id, "weight": weight}
	if reps > 0 {
		m["reps"] = reps
	}
	if dur > 0 {
		m["durationSec"] = dur
	}
	return m
}

func seedWorkouts(ids map[string]int64) []seedWorkout {
	blockID := 0
	nextID := func() string { blockID++; return fmt.Sprintf("seed-%d", blockID) }
	structure := func(blocks ...map[string]any) map[string]any {
		return map[string]any{"version": 1, "prepareSec": 10, "blockRestSec": 60, "blocks": blocks}
	}
	return []seedWorkout{
		{
			name:  "Swing & Squat Basics",
			notes: "Grundlagen-Session: Warm-up, schwere Swings, Goblet Squats, Plank.",
			structure: structure(
				map[string]any{"id": nextID(), "type": "circuit", "title": "Warm-up", "rounds": 2,
					"restBetweenItemsSec": 0, "restBetweenRoundsSec": 30,
					"items": []any{
						item(ids["Halo"], 12, 5, 0),
						item(ids["Goblet Squat"], 12, 5, 0),
						item(ids["Two-Hand Swing"], 12, 10, 0),
					}},
				map[string]any{"id": nextID(), "type": "sets", "title": "Swings", "sets": 5, "restSec": 45, "record": true,
					"item": item(ids["Two-Hand Swing"], 16, 15, 0)},
				map[string]any{"id": nextID(), "type": "sets", "title": "Goblet Squats", "sets": 4, "restSec": 60,
					"item": item(ids["Goblet Squat"], 16, 10, 0)},
				map[string]any{"id": nextID(), "type": "sets", "title": "Plank", "sets": 3, "restSec": 20,
					"item": item(ids["Plank"], "bw", 0, 40)},
			),
		},
		{
			name:  "Tabata Swings",
			notes: "8 Runden à 20 s Arbeit / 10 s Pause. Kurz und knackig.",
			structure: structure(
				map[string]any{"id": nextID(), "type": "interval", "title": "Tabata", "rounds": 8, "workSec": 20, "restSec": 10,
					"items": []any{item(ids["Two-Hand Swing"], 16, 0, 0)}},
			),
		},
		{
			name:  "EMOM 12: Clean, Press, Squat",
			notes: "Jede Minute eine Übung im Wechsel. Unilaterale Übungen wechseln automatisch die Seite.",
			structure: structure(
				map[string]any{"id": nextID(), "type": "emom", "title": "EMOM", "minutes": 12,
					"items": []any{
						item(ids["Clean"], 12, 5, 0),
						item(ids["Press"], 12, 5, 0),
						item(ids["Goblet Squat"], 16, 8, 0),
					}},
			),
		},
		{
			name:  "AMRAP 15",
			notes: "So viele Runden wie möglich in 15 Minuten.",
			structure: structure(
				map[string]any{"id": nextID(), "type": "amrap", "title": "AMRAP", "durationSec": 900,
					"items": []any{
						item(ids["Two-Hand Swing"], 16, 15, 0),
						item(ids["Goblet Squat"], 16, 10, 0),
						item(ids["Push-up"], "bw", 10, 0),
					}},
			),
		},
	}
}
