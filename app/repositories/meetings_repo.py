from __future__ import annotations

from sqlite3 import Row
from typing import Any

from app.db import Database
from app.utils.dates import now_str


class MeetingsRepository:
    def __init__(self, db: Database) -> None:
        self.db = db

    def create_meeting(
        self,
        *,
        title: str,
        description: str,
        source_type: str,
        owner_user_id: int,
    ) -> int:
        timestamp = now_str()
        with self.db.connect() as conn:
            cursor = conn.execute(
                '''
                INSERT INTO meetings (
                    title, description, source_type, status, owner_user_id,
                    started_at, ended_at, created_at, updated_at
                )
                VALUES (?, ?, ?, 'planned', ?, '', '', ?, ?)
                ''',
                (title, description, source_type, owner_user_id, timestamp, timestamp),
            )
            return int(cursor.lastrowid)

    def get_meeting(self, meeting_id: int) -> Row | None:
        with self.db.connect() as conn:
            return conn.execute(
                'SELECT * FROM meetings WHERE id = ?',
                (meeting_id,),
            ).fetchone()

    def list_meetings_by_owner(self, owner_user_id: int) -> list[Row]:
        with self.db.connect() as conn:
            return conn.execute(
                '''
                SELECT *
                FROM meetings
                WHERE owner_user_id = ?
                ORDER BY id DESC
                LIMIT 20
                ''',
                (owner_user_id,),
            ).fetchall()

    def start_meeting(self, meeting_id: int) -> None:
        timestamp = now_str()
        with self.db.connect() as conn:
            conn.execute(
                '''
                UPDATE meetings
                SET status = 'active', started_at = ?, updated_at = ?
                WHERE id = ?
                ''',
                (timestamp, timestamp, meeting_id),
            )

    def end_meeting(self, meeting_id: int) -> None:
        timestamp = now_str()
        with self.db.connect() as conn:
            conn.execute(
                '''
                UPDATE meetings
                SET status = 'finished', ended_at = ?, updated_at = ?
                WHERE id = ?
                ''',
                (timestamp, timestamp, meeting_id),
            )
