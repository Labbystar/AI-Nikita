from __future__ import annotations

from sqlite3 import Row

from app.repositories.items_repo import MeetingItemsRepository
from app.repositories.meetings_repo import MeetingsRepository
from app.repositories.participants_repo import ParticipantsRepository
from app.services.protocol_service import ProtocolService


class MeetingService:
    def __init__(
        self,
        meetings_repo: MeetingsRepository,
        participants_repo: ParticipantsRepository,
        items_repo: MeetingItemsRepository,
        protocol_service: ProtocolService,
    ) -> None:
        self.meetings_repo = meetings_repo
        self.participants_repo = participants_repo
        self.items_repo = items_repo
        self.protocol_service = protocol_service

    def create_meeting(
        self,
        *,
        title: str,
        description: str,
        source_type: str,
        owner_user_id: int,
        owner_display_name: str,
    ) -> int:
        meeting_id = self.meetings_repo.create_meeting(
            title=title,
            description=description,
            source_type=source_type,
            owner_user_id=owner_user_id,
        )
        self.participants_repo.add_participant(
            meeting_id=meeting_id,
            telegram_user_id=owner_user_id,
            display_name=owner_display_name,
            role='organizer',
        )
        return meeting_id

    def add_participant(self, meeting_id: int, display_name: str, telegram_user_id: int | None = None) -> int:
        return self.participants_repo.add_participant(
            meeting_id=meeting_id,
            telegram_user_id=telegram_user_id,
            display_name=display_name,
        )

    def add_simple_item(self, meeting_id: int, item_type: str, content: str, author_user_id: int | None) -> int:
        return self.items_repo.add_item(
            meeting_id=meeting_id,
            item_type=item_type,
            content=content,
            author_user_id=author_user_id,
        )

    def add_action(
        self,
        *,
        meeting_id: int,
        content: str,
        assigned_display_name: str,
        deadline_at: str,
        author_user_id: int | None,
    ) -> int:
        return self.items_repo.add_item(
            meeting_id=meeting_id,
            item_type='action',
            content=content,
            author_user_id=author_user_id,
            assigned_display_name=assigned_display_name,
            deadline_at=deadline_at,
        )

    def start_meeting(self, meeting_id: int) -> None:
        self.meetings_repo.start_meeting(meeting_id)

    def end_meeting(self, meeting_id: int) -> str:
        self.meetings_repo.end_meeting(meeting_id)
        meeting = self.meetings_repo.get_meeting(meeting_id)
        if meeting is None:
            raise ValueError('Meeting not found')
        protocol = self.protocol_service.build_protocol_text(meeting)
        self.protocol_service.save_protocol(meeting_id, protocol)
        return protocol

    def get_meeting(self, meeting_id: int) -> Row | None:
        return self.meetings_repo.get_meeting(meeting_id)

    def list_user_meetings(self, owner_user_id: int) -> list[Row]:
        return self.meetings_repo.list_meetings_by_owner(owner_user_id)

    def get_meeting_card_text(self, meeting_id: int) -> str:
        meeting = self.meetings_repo.get_meeting(meeting_id)
        if meeting is None:
            return 'Встреча не найдена.'

        participants = self.participants_repo.list_participants(meeting_id)
        notes_count = len(self.items_repo.list_items(meeting_id, 'note'))
        decisions_count = len(self.items_repo.list_items(meeting_id, 'decision'))
        actions_count = len(self.items_repo.list_items(meeting_id, 'action'))
        risks_count = len(self.items_repo.list_items(meeting_id, 'risk'))

        participants_text = '\n'.join(f"— {p['display_name']}" for p in participants) if participants else '— нет'

        return (
            f"📅 Встреча #{meeting['id']}\n"
            f"Название: {meeting['title']}\n"
            f"Источник: {meeting['source_type']}\n"
            f"Статус: {meeting['status']}\n\n"
            f"Участники:\n{participants_text}\n\n"
            f"Заметки: {notes_count}\n"
            f"Решения: {decisions_count}\n"
            f"Поручения: {actions_count}\n"
            f"Риски: {risks_count}"
        )
