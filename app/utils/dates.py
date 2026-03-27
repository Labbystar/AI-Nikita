from __future__ import annotations

from datetime import datetime


DB_DT_FORMAT = '%Y-%m-%d %H:%M:%S'
DISPLAY_DT_FORMAT = '%d.%m.%Y %H:%M'



def now_str() -> str:
    return datetime.now().strftime(DB_DT_FORMAT)



def format_dt(dt_str: str) -> str:
    if not dt_str:
        return '—'
    try:
        return datetime.strptime(dt_str, DB_DT_FORMAT).strftime(DISPLAY_DT_FORMAT)
    except ValueError:
        return dt_str
