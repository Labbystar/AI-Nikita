from aiogram import F, Router
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.types import CallbackQuery, Message

from app.keyboards.meetings import (
    meeting_actions_keyboard,
    meeting_card_keyboard,
    meetings_list_keyboard,
    source_type_keyboard,
)
from app.services.meeting_service import MeetingService
from app.states.meeting_states import (
    AddActionState,
    AddDecisionState,
    AddNoteState,
    AddParticipantState,
    AddRiskState,
    CreateMeetingState,
)

router = Router()

_MEETING_SERVICE: MeetingService | None = None


def set_meeting_service(service: MeetingService) -> None:
    global _MEETING_SERVICE
    _MEETING_SERVICE = service


def get_service(message_or_callback: Message | CallbackQuery) -> MeetingService:
    if _MEETING_SERVICE is None:
        raise RuntimeError("MeetingService is not initialized")
    return _MEETING_SERVICE


@router.message(Command("new_meeting"))
async def cmd_new_meeting(message: Message, state: FSMContext) -> None:
    await state.clear()
    await state.set_state(CreateMeetingState.title)
    await message.answer("Введите название встречи:")


@router.message(CreateMeetingState.title)
async def create_meeting_title(message: Message, state: FSMContext) -> None:
    await state.update_data(title=message.text.strip())
    await state.set_state(CreateMeetingState.description)
    await message.answer("Введите краткое описание встречи:")


@router.message(CreateMeetingState.description)
async def create_meeting_description(message: Message, state: FSMContext) -> None:
    await state.update_data(description=message.text.strip())
    await state.set_state(CreateMeetingState.source_type)
    await message.answer(
        "Выберите тип встречи:",
        reply_markup=source_type_keyboard(),
    )


@router.callback_query(F.data.startswith("source:"))
async def create_meeting_source(callback: CallbackQuery, state: FSMContext) -> None:
    source_type = callback.data.split(":", 1)[1]
    await state.update_data(source_type=source_type)

    data = await state.get_data()
    service = get_service(callback)

    meeting_id = service.create_meeting(
        owner_user_id=callback.from_user.id,
        title=data["title"],
        description=data["description"],
        source_type=data["source_type"],
    )

    await state.clear()
    await callback.message.answer(
        f"Встреча создана: #{meeting_id}\n\n"
        f"Название: {data['title']}\n"
        f"Тип: {data['source_type']}"
    )
    await callback.message.answer(
        "Открой карточку встречи:",
        reply_markup=meeting_card_keyboard(meeting_id),
    )
    await callback.answer()


@router.message(Command("my_meetings"))
async def cmd_my_meetings(message: Message) -> None:
    service = get_service(message)
    meetings = service.list_user_meetings(message.from_user.id)

    if not meetings:
        await message.answer("У вас пока нет встреч.")
        return

    await message.answer(
        "Ваши встречи:",
        reply_markup=meetings_list_keyboard(meetings),
    )


@router.callback_query(F.data.startswith("meeting:"))
async def open_meeting(callback: CallbackQuery) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    service = get_service(callback)
    card_text = service.get_meeting_card_text(meeting_id)

    await callback.message.answer(
        card_text,
        reply_markup=meeting_actions_keyboard(meeting_id),
    )
    await callback.answer()


@router.callback_query(F.data.startswith("start_meeting:"))
async def start_meeting(callback: CallbackQuery) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    service = get_service(callback)
    service.start_meeting(meeting_id)

    await callback.message.answer(
        f"Встреча #{meeting_id} переведена в статус active.",
        reply_markup=meeting_actions_keyboard(meeting_id),
    )
    await callback.answer()


@router.callback_query(F.data.startswith("end_meeting:"))
async def end_meeting(callback: CallbackQuery) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    service = get_service(callback)
    protocol_text = service.end_meeting_and_build_protocol(meeting_id)

    await callback.message.answer(protocol_text)
    await callback.answer()


@router.callback_query(F.data.startswith("add_participant:"))
async def add_participant_start(callback: CallbackQuery, state: FSMContext) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    await state.clear()
    await state.set_state(AddParticipantState.name)
    await state.update_data(meeting_id=meeting_id)

    await callback.message.answer(
        "Введите имя участника, которого нужно добавить:"
    )
    await callback.answer()


@router.message(AddParticipantState.name)
async def add_participant_finish(message: Message, state: FSMContext) -> None:
    data = await state.get_data()
    meeting_id = data["meeting_id"]
    service = get_service(message)

    service.add_participant(
        meeting_id=meeting_id,
        telegram_user_id=None,
        display_name=message.text.strip(),
        role="participant",
    )

    await state.clear()
    await message.answer(
        f"Участник добавлен во встречу #{meeting_id}.",
        reply_markup=meeting_actions_keyboard(meeting_id),
    )


@router.callback_query(F.data.startswith("add_note:"))
async def add_note_start(callback: CallbackQuery, state: FSMContext) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    await state.clear()
    await state.set_state(AddNoteState.text)
    await state.update_data(meeting_id=meeting_id)

    await callback.message.answer("Введите заметку:")
    await callback.answer()


@router.message(AddNoteState.text)
async def add_note_finish(message: Message, state: FSMContext) -> None:
    data = await state.get_data()
    meeting_id = data["meeting_id"]
    service = get_service(message)

    service.add_item(
        meeting_id=meeting_id,
        item_type="note",
        content=message.text.strip(),
        author_user_id=message.from_user.id,
    )

    await state.clear()
    await message.answer(
        "Заметка сохранена.",
        reply_markup=meeting_actions_keyboard(meeting_id),
    )


@router.callback_query(F.data.startswith("add_decision:"))
async def add_decision_start(callback: CallbackQuery, state: FSMContext) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    await state.clear()
    await state.set_state(AddDecisionState.text)
    await state.update_data(meeting_id=meeting_id)

    await callback.message.answer("Введите решение:")
    await callback.answer()


@router.message(AddDecisionState.text)
async def add_decision_finish(message: Message, state: FSMContext) -> None:
    data = await state.get_data()
    meeting_id = data["meeting_id"]
    service = get_service(message)

    service.add_item(
        meeting_id=meeting_id,
        item_type="decision",
        content=message.text.strip(),
        author_user_id=message.from_user.id,
    )

    await state.clear()
    await message.answer(
        "Решение сохранено.",
        reply_markup=meeting_actions_keyboard(meeting_id),
    )


@router.callback_query(F.data.startswith("add_risk:"))
async def add_risk_start(callback: CallbackQuery, state: FSMContext) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    await state.clear()
    await state.set_state(AddRiskState.text)
    await state.update_data(meeting_id=meeting_id)

    await callback.message.answer("Введите риск / блокер:")
    await callback.answer()


@router.message(AddRiskState.text)
async def add_risk_finish(message: Message, state: FSMContext) -> None:
    data = await state.get_data()
    meeting_id = data["meeting_id"]
    service = get_service(message)

    service.add_item(
        meeting_id=meeting_id,
        item_type="risk",
        content=message.text.strip(),
        author_user_id=message.from_user.id,
    )

    await state.clear()
    await message.answer(
        "Риск сохранён.",
        reply_markup=meeting_actions_keyboard(meeting_id),
    )


@router.callback_query(F.data.startswith("add_action:"))
async def add_action_start(callback: CallbackQuery, state: FSMContext) -> None:
    meeting_id = int(callback.data.split(":", 1)[1])
    await state.clear()
    await state.set_state(AddActionState.text)
    await state.update_data(meeting_id=meeting_id)

    await callback.message.answer(
        "Введите поручение в формате:\n\n"
        "Исполнитель | Что сделать | Срок\n\n"
        "Пример:\n"
        "Иван | Подготовить финмодель | 30.03.2026 18:00"
    )
    await callback.answer()


@router.message(AddActionState.text)
async def add_action_finish(message: Message, state: FSMContext) -> None:
    raw_text = message.text.strip()
    parts = [part.strip() for part in raw_text.split("|")]

    if len(parts) != 3:
        await message.answer(
            "Неверный формат.\n\n"
            "Используйте:\n"
            "Исполнитель | Что сделать | Срок"
        )
        return

    assignee, content, deadline = parts
    data = await state.get_data()
    meeting_id = data["meeting_id"]
    service = get_service(message)

    service.add_item(
        meeting_id=meeting_id,
        item_type="action",
        content=content,
        author_user_id=message.from_user.id,
        assigned_display_name=assignee,
        deadline_at=deadline,
    )

    await state.clear()
    await message.answer(
        "Поручение сохранено.",
        reply_markup=meeting_actions_keyboard(meeting_id),
    )
