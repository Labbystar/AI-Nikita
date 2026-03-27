import os
import sqlite3

from app.config import settings


def get_connection() -> sqlite3.Connection:
    db_dir = os.path.dirname(settings.db_path)
    if db_dir:
        os.makedirs(db_dir, exist_ok=True)

    conn = sqlite3.connect(settings.db_path)
    conn.row_factory = sqlite3.Row
    return conn


def init_db() -> None:
    conn = get_connection()
    cursor = conn.cursor()

    cursor.execute("""
    CREATE TABLE IF NOT EXISTS meetings (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        description TEXT DEFAULT '',
        source_type TEXT NOT NULL DEFAULT 'manual',
        status TEXT NOT NULL DEFAULT 'planned',
        owner_user_id INTEGER NOT NULL,
        started_at TEXT DEFAULT '',
        ended_at TEXT DEFAULT '',
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL
    )
    """)

    cursor.execute("""
    CREATE TABLE IF NOT EXISTS participants (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        meeting_id INTEGER NOT NULL,
        telegram_user_id INTEGER,
        display_name TEXT NOT NULL,
        role TEXT DEFAULT 'participant',
        created_at TEXT NOT NULL,
        FOREIGN KEY (meeting_id) REFERENCES meetings(id)
    )
    """)

    cursor.execute("""
    CREATE TABLE IF NOT EXISTS meeting_items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        meeting_id INTEGER NOT NULL,
        item_type TEXT NOT NULL,
        content TEXT NOT NULL,
        author_user_id INTEGER,
        assigned_user_id INTEGER,
        assigned_display_name TEXT DEFAULT '',
        deadline_at TEXT DEFAULT '',
        status TEXT NOT NULL DEFAULT 'open',
        confidence REAL DEFAULT 1.0,
        source_kind TEXT DEFAULT 'manual',
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL,
        FOREIGN KEY (meeting_id) REFERENCES meetings(id)
    )
    """)

    cursor.execute("""
    CREATE TABLE IF NOT EXISTS artifacts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        meeting_id INTEGER NOT NULL,
        artifact_type TEXT NOT NULL,
        file_path TEXT DEFAULT '',
        text_content TEXT DEFAULT '',
        mime_type TEXT DEFAULT '',
        created_at TEXT NOT NULL,
        FOREIGN KEY (meeting_id) REFERENCES meetings(id)
    )
    """)

    cursor.execute("""
    CREATE TABLE IF NOT EXISTS protocols (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        meeting_id INTEGER NOT NULL,
        protocol_text TEXT NOT NULL,
        summary_text TEXT DEFAULT '',
        created_at TEXT NOT NULL,
        FOREIGN KEY (meeting_id) REFERENCES meetings(id)
    )
    """)

    conn.commit()
    conn.close()
