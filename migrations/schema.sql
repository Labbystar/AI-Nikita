CREATE TABLE IF NOT EXISTS meetings (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_url TEXT,
    source_meeting_id TEXT,
    status TEXT NOT NULL,
    transcript TEXT,
    protocol_text TEXT,
    summary_text TEXT,
    started_at DATETIME,
    ended_at DATETIME,
    created_by_telegram_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_meetings_created_by ON meetings(created_by_telegram_id);
CREATE INDEX IF NOT EXISTS idx_meetings_source_meeting_id ON meetings(source_meeting_id);

CREATE TABLE IF NOT EXISTS meeting_participants (
    id TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    display_name TEXT NOT NULL,
    telegram_user_id INTEGER,
    role TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(meeting_id) REFERENCES meetings(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS meeting_artifacts (
    id TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    artifact_type TEXT NOT NULL,
    file_id TEXT,
    file_url TEXT,
    text_content TEXT,
    mime_type TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(meeting_id) REFERENCES meetings(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS meeting_items (
    id TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    item_type TEXT NOT NULL,
    content TEXT NOT NULL,
    assigned_to TEXT,
    deadline TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    confidence REAL NOT NULL DEFAULT 1.0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(meeting_id) REFERENCES meetings(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    meeting_id TEXT,
    title TEXT NOT NULL,
    description TEXT,
    assigned_to TEXT,
    deadline TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(meeting_id) REFERENCES meetings(id) ON DELETE SET NULL
);
