from __future__ import annotations

from sqlite3 import Row

from app.db import Database
from app.utils.dates import now_str


class MeetingItemsRepository:
    def __init__(self, db: Database) -> None:
        self.db = db

    def add_item(
        self,
        *,
        meeting_id: int,
        item_type: str,
        content: str,
        author_user_id: int | None,
        assigned_user_id: int | None = None,
        assigned_display_name: str = '',
        deadline_at: str = '',
        status: str = 'open',
        confidence: float = 1.0,
        source_kind: str = 'manual',
    ) -> int:
        timestamp = now_str()
        with self.db.connect() as conn:
            cursor = conn.execute(
                '''
                INSERT INTO meeting_items (
                    meeting_id, item_type, content, author_user_id,
                    assigned_user_id, assigned_display_name, deadline_at,
                    status, confidence, source_kind, created_at, updated_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ''',
                (
                    meeting_id,
                    item_type,
                    content,
                    author_user_id,
                    assigned_user_id,
                    assigned_display_name,
                    deadline_at,
                    status,
                    confidence,
                    source_kind,
                    timestamp,
                    timestamp,
                ),
            )
            return int(cursor.lastrowid)

    def list_items(self, meeting_id: int, item_type: str | None = None) -> list[Row]:
        query = 'SELECT * FROM meeting_items WHERE meeting_id = ?'
        params: list[object] = [meeting_id]
        if item_type:
            query += ' AND item_type = ?'
            params.append(item_type)
        query += ' ORDER BY id ASC'
        with self.db.connect() as conn:
            return conn.execute(query, params).fetchall()
