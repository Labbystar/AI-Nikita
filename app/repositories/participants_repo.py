from __future__ import annotations

from sqlite3 import Row

from app.db import Database
from app.utils.dates import now_str


class ParticipantsRepository:
    def __init__(self, db: Database) -> None:
        self.db = db

    def add_participant(
        self,
        *,
        meeting_id: int,
        display_name: str,
        telegram_user_id: int | None = None,
        role: str = 'participant',
    ) -> int:
        with self.db.connect() as conn:
            cursor = conn.execute(
                '''
                INSERT INTO participants (
                    meeting_id, telegram_user_id, display_name, role, created_at
                )
                VALUES (?, ?, ?, ?, ?)
                ''',
                (meeting_id, telegram_user_id, display_name, role, now_str()),
            )
            return int(cursor.lastrowid)

    def list_participants(self, meeting_id: int) -> list[Row]:
        with self.db.connect() as conn:
            return conn.execute(
                '''
                SELECT *
                FROM participants
                WHERE meeting_id = ?
                ORDER BY id ASC
                ''',
                (meeting_id,),
            ).fetchall()
