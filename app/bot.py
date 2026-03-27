from __future__ import annotations

import asyncio
import logging

from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from aiogram.enums import ParseMode

from app.config import load_config
from app.db import Database
from app.handlers.meetings import router as meetings_router
from app.handlers.start import router as start_router
from app.repositories.items_repo import MeetingItemsRepository
from app.repositories.meetings_repo import MeetingsRepository
from app.repositories.participants_repo import ParticipantsRepository
from app.services.meeting_service import MeetingService
from app.services.protocol_service import ProtocolService


async def main() -> None:
    logging.basicConfig(level=logging.INFO)

    config = load_config()
    db = Database(config.db_path)
    db.init()

    bot = Bot(
        token=config.bot_token,
        default=DefaultBotProperties(parse_mode=ParseMode.HTML),
    )
    dp = Dispatcher()

    meetings_repo = MeetingsRepository(db)
    participants_repo = ParticipantsRepository(db)
    items_repo = MeetingItemsRepository(db)
    protocol_service = ProtocolService(participants_repo, items_repo, db)
    meeting_service = MeetingService(meetings_repo, participants_repo, items_repo, protocol_service)

    bot['meeting_service'] = meeting_service

    dp.include_router(start_router)
    dp.include_router(meetings_router)

    await bot.delete_webhook(drop_pending_updates=True)
    await dp.start_polling(bot)


if __name__ == '__main__':
    asyncio.run(main())
