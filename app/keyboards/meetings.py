from aiogram.types import InlineKeyboardButton, InlineKeyboardMarkup
from aiogram.utils.keyboard import InlineKeyboardBuilder


SOURCE_TYPES = {
    'manual': 'Ручной режим',
    'offline': 'Оффлайн',
    'zoom': 'Zoom',
    'telemost': 'Яндекс Телемост',
}



def source_type_keyboard() -> InlineKeyboardMarkup:
    builder = InlineKeyboardBuilder()
    for source_key, label in SOURCE_TYPES.items():
        builder.button(text=label, callback_data=f'source:{source_key}')
    builder.adjust(2)
    return builder.as_markup()



def meeting_actions_keyboard(meeting_id: int, status: str) -> InlineKeyboardMarkup:
    builder = InlineKeyboardBuilder()

    builder.button(text='Участник', callback_data=f'meeting:add_participant:{meeting_id}')
    builder.button(text='Заметка', callback_data=f'meeting:add_note:{meeting_id}')
    builder.button(text='Решение', callback_data=f'meeting:add_decision:{meeting_id}')
    builder.button(text='Поручение', callback_data=f'meeting:add_action:{meeting_id}')
    builder.button(text='Риск', callback_data=f'meeting:add_risk:{meeting_id}')

    if status == 'planned':
        builder.button(text='Начать встречу', callback_data=f'meeting:start:{meeting_id}')
    elif status == 'active':
        builder.button(text='Завершить встречу', callback_data=f'meeting:end:{meeting_id}')

    builder.button(text='Обновить карточку', callback_data=f'meeting:view:{meeting_id}')
    builder.adjust(2)
    return builder.as_markup()
