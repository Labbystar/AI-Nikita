import asyncio
import logging

from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from aiogram.enums import ParseMode

from app.config import settings
from app.db import init_db
from app.handlers.meetings import router as meetings_router, set_meeting_service
from app.handlers.start import router as start_router
from app.services.meeting_service import MeetingService


async def main() -> None:
    logging.basicConfig(level=logging.INFO)

    init_db()

    bot = Bot(
        token=settings.bot_token,
        default=DefaultBotProperties(parse_mode=ParseMode.HTML),
    )

    dp = Dispatcher()

    meeting_service = MeetingService()

    set_meeting_service(meeting_service)

    dp.include_router(start_router)
    dp.include_router(meetings_router)

    await bot.delete_webhook(drop_pending_updates=True)
    await dp.start_polling(bot)


if __name__ == "__main__":
    asyncio.run(main())
