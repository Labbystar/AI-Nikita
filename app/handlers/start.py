from aiogram import Router
from aiogram.filters import CommandStart
from aiogram.types import Message

from app.keyboards.main_menu import main_menu_keyboard

router = Router()


@router.message(CommandStart())
async def cmd_start(message: Message) -> None:
    text = (
        'Это Meeting Assistant Bot.\n\n'
        'Что умеет MVP:\n'
        '— создавать встречи\n'
        '— добавлять участников\n'
        '— фиксировать заметки, решения, поручения и риски\n'
        '— формировать итоговый протокол\n\n'
        'Нажми «Создать встречу» или используй /new_meeting.'
    )
    await message.answer(text, reply_markup=main_menu_keyboard())
