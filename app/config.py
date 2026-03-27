from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv

BASE_DIR = Path(__file__).resolve().parent.parent
load_dotenv(BASE_DIR / '.env')


@dataclass(slots=True)
class Config:
    bot_token: str
    db_path: Path
    default_timezone: str



def load_config() -> Config:
    bot_token = os.getenv('BOT_TOKEN', '').strip()
    db_path_raw = os.getenv('DB_PATH', '/data/meeting_assistant.db').strip()
    default_timezone = os.getenv('DEFAULT_TIMEZONE', 'Europe/Amsterdam').strip()

    if not bot_token:
        raise RuntimeError('BOT_TOKEN is not set. Copy .env.example to .env and fill it in.')

    db_path = (BASE_DIR / db_path_raw).resolve()
    db_path.parent.mkdir(parents=True, exist_ok=True)

    return Config(
        bot_token=bot_token,
        db_path=db_path,
        default_timezone=default_timezone,
    )
