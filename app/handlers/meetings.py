from __future__ import annotations

from aiogram import F, Router
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.types import CallbackQuery, Message

from app.keyboards.main_menu import main_menu_keyboard
from app.keyboards.meetings import SOURCE_TYPES, meeting_actions_keyboard, source_type_keyboard
from app.services.meeting_service import MeetingService
from app.states.meeting_states import AddActionState, AddItemState, AddParticipantState, CreateMeetingState

router = Router()


def get_service(message_or_callback: Message | CallbackQuery) -> MeetingService:
    return message_or_callback.bot['meeting_service']


@router.message(Command('new_meeting'))
@router.message(F.text == 'Создать встречу')
async def new_meeting(message: Message, state: FSMContext) -> None:
    await state.clear()
    await state.set_state(CreateMeetingState.waiting_for_title)
    await message.answer('Введи название встречи.')


@router.message(CreateMeetingState.waiting_for_title)
async def meeting_title(message: Message, state: FSMContext) -> None:
    if not message.text:
        await message.answer('Нужно текстом отправить название встречи.')
        return
    await state.update_data(title=message.text.strip())
    await state.set_state(CreateMeetingState.waiting_for_description)
    await message.answer('Теперь отправь краткое описание встречи. Если не нужно, напиши: -')


@router.message(CreateMeetingState.waiting_for_description)
async def meeting_description(message: Message, state: FSMContext) -> None:
    description = (message.text or '').strip()
    if description == '-':
        description = ''
    await state.update_data(description=description)
    await state.set_state(CreateMeetingState.waiting_for_source_type)
    await message.answer('Выбери источник встречи.', reply_markup=source_type_keyboard())


@router.callback_query(CreateMeetingState.waiting_for_source_type, F.data.startswith('source:'))
async def meeting_source(callback: CallbackQuery, state: FSMContext) -> None:
    service = get_service(callback)
    source_type = callback.data.split(':', 1)[1]
    data = await state.get_data()
    user = callback.from_user
    owner_display_name = user.full_name or user.username or str(user.id)

    meeting_id = service.create_meeting(
        title=data['title'],
        description=data.get('description', ''),
        source_type=source_type,
        owner_user_id=user.id,
        owner_display_name=owner_display_name,
    )
    await state.clear()
    meeting = service.get_meeting(meeting_id)
    assert meeting is not None
    await callback.message.answer(
        f"Встреча создана.\n\nID: {meeting_id}\nНазвание: {meeting['title']}\nИсточник: {SOURCE_TYPES.get(source_type, source_type)}",
        reply_markup=main_menu_keyboard(),
    )
    await callback.message.answer(
        service.get_meeting_card_text(meeting_id),
        reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
    )
    await callback.answer()


@router.message(Command('my_meetings'))
@router.message(F.text == 'Мои встречи')
async def my_meetings(message: Message) -> None:
    service = get_service(message)
    meetings = service.list_user_meetings(message.from_user.id)
    if not meetings:
        await message.answer('У тебя пока нет встреч.')
        return

    lines = ['Твои последние встречи:']
    for row in meetings:
        lines.append(f"— #{row['id']} | {row['title']} | {row['status']} | /meeting_{row['id']}")
    await message.answer('\n'.join(lines))


@router.message(F.text.regexp(r'^/meeting_\d+$'))
async def open_meeting(message: Message) -> None:
    service = get_service(message)
    meeting_id = int((message.text or '').split('_')[1])
    meeting = service.get_meeting(meeting_id)
    if meeting is None:
        await message.answer('Встреча не найдена.')
        return
    await message.answer(
        service.get_meeting_card_text(meeting_id),
        reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
    )


@router.callback_query(F.data.startswith('meeting:view:'))
async def view_meeting(callback: CallbackQuery) -> None:
    service = get_service(callback)
    meeting_id = int(callback.data.split(':')[-1])
    meeting = service.get_meeting(meeting_id)
    if meeting is None:
        await callback.answer('Встреча не найдена.', show_alert=True)
        return
    await callback.message.edit_text(
        service.get_meeting_card_text(meeting_id),
        reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
    )
    await callback.answer()


@router.callback_query(F.data.startswith('meeting:start:'))
@router.message(F.text.regexp(r'^/start_meeting_\d+$'))
async def start_meeting(event: CallbackQuery | Message) -> None:
    service = get_service(event)
    if isinstance(event, CallbackQuery):
        meeting_id = int(event.data.split(':')[-1])
    else:
        meeting_id = int((event.text or '').split('_')[-1])

    meeting = service.get_meeting(meeting_id)
    if meeting is None:
        if isinstance(event, CallbackQuery):
            await event.answer('Встреча не найдена.', show_alert=True)
        else:
            await event.answer('Встреча не найдена.')
        return

    service.start_meeting(meeting_id)
    meeting = service.get_meeting(meeting_id)
    assert meeting is not None

    if isinstance(event, CallbackQuery):
        await event.message.edit_text(
            service.get_meeting_card_text(meeting_id),
            reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
        )
        await event.answer('Встреча начата.')
    else:
        await event.answer(
            'Встреча начата. Теперь можно добавлять заметки, решения, поручения и риски.',
            reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
        )


@router.callback_query(F.data.startswith('meeting:end:'))
@router.message(F.text.regexp(r'^/end_meeting_\d+$'))
async def end_meeting(event: CallbackQuery | Message) -> None:
    service = get_service(event)
    if isinstance(event, CallbackQuery):
        meeting_id = int(event.data.split(':')[-1])
    else:
        meeting_id = int((event.text or '').split('_')[-1])

    if service.get_meeting(meeting_id) is None:
        if isinstance(event, CallbackQuery):
            await event.answer('Встреча не найдена.', show_alert=True)
        else:
            await event.answer('Встреча не найдена.')
        return

    protocol = service.end_meeting(meeting_id)
    meeting = service.get_meeting(meeting_id)
    assert meeting is not None

    if isinstance(event, CallbackQuery):
        await event.message.answer('Встреча завершена. Ниже итоговый протокол:')
        await event.message.answer(protocol)
        await event.message.answer(
            service.get_meeting_card_text(meeting_id),
            reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
        )
        await event.answer()
    else:
        await event.answer('Встреча завершена. Ниже итоговый протокол:')
        await event.answer(protocol)


@router.callback_query(F.data.startswith('meeting:add_participant:'))
async def add_participant_start(callback: CallbackQuery, state: FSMContext) -> None:
    meeting_id = int(callback.data.split(':')[-1])
    await state.set_state(AddParticipantState.waiting_for_name)
    await state.update_data(meeting_id=meeting_id)
    await callback.message.answer('Введи имя участника.')
    await callback.answer()


@router.message(AddParticipantState.waiting_for_name)
async def add_participant_finish(message: Message, state: FSMContext) -> None:
    service = get_service(message)
    data = await state.get_data()
    meeting_id = int(data['meeting_id'])
    display_name = (message.text or '').strip()
    if not display_name:
        await message.answer('Имя участника не может быть пустым.')
        return
    service.add_participant(meeting_id, display_name)
    await state.clear()
    meeting = service.get_meeting(meeting_id)
    assert meeting is not None
    await message.answer('Участник добавлен.')
    await message.answer(
        service.get_meeting_card_text(meeting_id),
        reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
    )


@router.callback_query(F.data.startswith('meeting:add_note:'))
async def add_note_start(callback: CallbackQuery, state: FSMContext) -> None:
    await _start_simple_item(callback, state, 'note', 'Отправь текст заметки.')


@router.callback_query(F.data.startswith('meeting:add_decision:'))
async def add_decision_start(callback: CallbackQuery, state: FSMContext) -> None:
    await _start_simple_item(callback, state, 'decision', 'Отправь текст решения.')


@router.callback_query(F.data.startswith('meeting:add_risk:'))
async def add_risk_start(callback: CallbackQuery, state: FSMContext) -> None:
    await _start_simple_item(callback, state, 'risk', 'Отправь текст риска или блокера.')


async def _start_simple_item(callback: CallbackQuery, state: FSMContext, item_type: str, prompt: str) -> None:
    meeting_id = int(callback.data.split(':')[-1])
    await state.set_state(AddItemState.waiting_for_content)
    await state.update_data(meeting_id=meeting_id, item_type=item_type)
    await callback.message.answer(prompt)
    await callback.answer()


@router.message(AddItemState.waiting_for_content)
async def add_simple_item_finish(message: Message, state: FSMContext) -> None:
    service = get_service(message)
    data = await state.get_data()
    content = (message.text or '').strip()
    if not content:
        await message.answer('Текст не может быть пустым.')
        return
    meeting_id = int(data['meeting_id'])
    item_type = str(data['item_type'])
    service.add_simple_item(meeting_id, item_type, content, message.from_user.id)
    await state.clear()
    meeting = service.get_meeting(meeting_id)
    assert meeting is not None
    labels = {'note': 'Заметка', 'decision': 'Решение', 'risk': 'Риск'}
    await message.answer(f"{labels.get(item_type, 'Запись')} добавлен(а).")
    await message.answer(
        service.get_meeting_card_text(meeting_id),
        reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
    )


@router.callback_query(F.data.startswith('meeting:add_action:'))
async def add_action_start(callback: CallbackQuery, state: FSMContext) -> None:
    meeting_id = int(callback.data.split(':')[-1])
    await state.set_state(AddActionState.waiting_for_action)
    await state.update_data(meeting_id=meeting_id)
    await callback.message.answer(
        'Отправь поручение в формате:\n'
        'Исполнитель | Что сделать | Срок\n\n'
        'Пример:\n'
        'Иван | Подготовить финмодель | 30.03.2026 18:00'
    )
    await callback.answer()


@router.message(AddActionState.waiting_for_action)
async def add_action_finish(message: Message, state: FSMContext) -> None:
    service = get_service(message)
    raw_text = (message.text or '').strip()
    parts = [part.strip() for part in raw_text.split('|')]
    if len(parts) != 3 or not all(parts):
        await message.answer('Неверный формат. Используй: Исполнитель | Что сделать | Срок')
        return

    assignee, content, deadline = parts
    data = await state.get_data()
    meeting_id = int(data['meeting_id'])
    service.add_action(
        meeting_id=meeting_id,
        content=content,
        assigned_display_name=assignee,
        deadline_at=deadline,
        author_user_id=message.from_user.id,
    )
    await state.clear()
    meeting = service.get_meeting(meeting_id)
    assert meeting is not None
    await message.answer('Поручение добавлено.')
    await message.answer(
        service.get_meeting_card_text(meeting_id),
        reply_markup=meeting_actions_keyboard(meeting_id, meeting['status']),
    )
