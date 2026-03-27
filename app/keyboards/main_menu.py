from aiogram.types import KeyboardButton, ReplyKeyboardMarkup


def main_menu_keyboard() -> ReplyKeyboardMarkup:
    return ReplyKeyboardMarkup(
        keyboard=[
            [KeyboardButton(text='Создать встречу')],
            [KeyboardButton(text='Мои встречи')],
        ],
        resize_keyboard=True,
    )
