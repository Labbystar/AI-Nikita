from __future__ import annotations

from sqlite3 import Row

from app.repositories.items_repo import MeetingItemsRepository
from app.repositories.participants_repo import ParticipantsRepository
from app.utils.dates import format_dt, now_str


class ProtocolService:
    def __init__(
        self,
        participants_repo: ParticipantsRepository,
        items_repo: MeetingItemsRepository,
        db,
    ) -> None:
        self.participants_repo = participants_repo
        self.items_repo = items_repo
        self.db = db

    def build_protocol_text(self, meeting: Row) -> str:
        participants = self.participants_repo.list_participants(int(meeting['id']))
        notes = self.items_repo.list_items(int(meeting['id']), 'note')
        decisions = self.items_repo.list_items(int(meeting['id']), 'decision')
        actions = self.items_repo.list_items(int(meeting['id']), 'action')
        risks = self.items_repo.list_items(int(meeting['id']), 'risk')

        participants_text = self._render_participants(participants)
        notes_text = self._render_simple_items(notes)
        decisions_text = self._render_simple_items(decisions)
        actions_text = self._render_actions(actions)
        risks_text = self._render_simple_items(risks)

        protocol_text = (
            f"📌 Протокол встречи: {meeting['title']}\n\n"
            f"Статус: завершена\n"
            f"Источник: {meeting['source_type']}\n"
            f"Дата начала: {format_dt(meeting['started_at'])}\n"
            f"Дата окончания: {format_dt(meeting['ended_at'])}\n\n"
            f"Участники:\n{participants_text}\n\n"
            f"Заметки:\n{notes_text}\n\n"
            f"Решения:\n{decisions_text}\n\n"
            f"Поручения:\n{actions_text}\n\n"
            f"Риски:\n{risks_text}"
        )
        return protocol_text

    def save_protocol(self, meeting_id: int, protocol_text: str, summary_text: str = '') -> int:
        with self.db.connect() as conn:
            cursor = conn.execute(
                '''
                INSERT INTO protocols (meeting_id, protocol_text, summary_text, created_at)
                VALUES (?, ?, ?, ?)
                ''',
                (meeting_id, protocol_text, summary_text, now_str()),
            )
            return int(cursor.lastrowid)

    @staticmethod
    def _render_participants(participants: list[Row]) -> str:
        if not participants:
            return '— участников пока нет'
        return '\n'.join(f"— {row['display_name']}" for row in participants)

    @staticmethod
    def _render_simple_items(items: list[Row]) -> str:
        if not items:
            return '— нет'
        return '\n'.join(f"{idx}. {row['content']}" for idx, row in enumerate(items, start=1))

    @staticmethod
    def _render_actions(items: list[Row]) -> str:
        if not items:
            return '— нет'
        lines: list[str] = []
        for idx, row in enumerate(items, start=1):
            assignee = row['assigned_display_name'] or 'не указан'
            deadline = row['deadline_at'] or 'без срока'
            lines.append(f"{idx}. {assignee} — {row['content']} — срок: {deadline}")
        return '\n'.join(lines)
