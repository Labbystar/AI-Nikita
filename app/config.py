import os
from dataclasses import dataclass

from dotenv import load_dotenv

load_dotenv()


@dataclass
class Settings:
    bot_token: str
    db_path: str
    default_timezone: str


def load_settings() -> Settings:
    bot_token = os.getenv("BOT_TOKEN", "").strip()
    db_path = os.getenv("DB_PATH", "/data/meeting_assistant.db").strip()
    default_timezone = os.getenv("DEFAULT_TIMEZONE", "Europe/Amsterdam").strip()

    if not bot_token:
        raise RuntimeError("BOT_TOKEN is not set")

    return Settings(
        bot_token=bot_token,
        db_path=db_path,
        default_timezone=default_timezone,
    )


settings = load_settings()
