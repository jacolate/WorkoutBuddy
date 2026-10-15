CREATE TABLE exercises (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    name              TEXT    NOT NULL,
    description       TEXT    NOT NULL DEFAULT '',
    category          TEXT    NOT NULL DEFAULT 'other',
    unilateral        INTEGER NOT NULL DEFAULT 0,
    default_weight_kg REAL,
    pose_profile      TEXT,
    cues              TEXT    NOT NULL DEFAULT '',
    camera_hint       TEXT    NOT NULL DEFAULT '',
    created_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE workouts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    notes      TEXT NOT NULL DEFAULT '',
    structure  TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE sessions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    workout_id   INTEGER REFERENCES workouts(id) ON DELETE SET NULL,
    workout_name TEXT NOT NULL DEFAULT '',
    started_at   TEXT NOT NULL,
    finished_at  TEXT,
    snapshot     TEXT NOT NULL,
    log          TEXT NOT NULL DEFAULT '[]',
    notes        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX sessions_started_at ON sessions(started_at DESC);

CREATE TABLE recordings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   INTEGER REFERENCES sessions(id) ON DELETE SET NULL,
    exercise_id  INTEGER REFERENCES exercises(id) ON DELETE SET NULL,
    filename     TEXT    NOT NULL,
    mime         TEXT    NOT NULL,
    size_bytes   INTEGER NOT NULL DEFAULT 0,
    duration_sec REAL    NOT NULL DEFAULT 0,
    label        TEXT    NOT NULL DEFAULT '',
    note         TEXT    NOT NULL DEFAULT '',
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX recordings_session ON recordings(session_id);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE auth_sessions (
    token_hash TEXT PRIMARY KEY,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
