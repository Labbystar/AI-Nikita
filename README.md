# Meeting Assistant Bot

Telegram-бот для фиксации встреч, участников, решений, поручений, рисков и генерации текстового протокола.

## Что умеет MVP
- создать встречу
- посмотреть список своих встреч
- открыть карточку встречи
- добавить участника
- начать встречу
- добавить заметку
- добавить решение
- добавить поручение
- добавить риск
- завершить встречу
- автоматически собрать текстовый протокол

## Стек
- Python 3.11+
- aiogram 3
- SQLite
- Railway

## Локальный запуск

```bash
python -m venv .venv
source .venv/bin/activate  # Windows: .venv\Scripts\activate
pip install -r requirements.txt
cp .env.example .env
python run.py
```

## Переменные окружения
Скопируй `.env.example` в `.env` и укажи свои значения:

```env
BOT_TOKEN=your_telegram_bot_token_here
DB_PATH=/data/meeting_assistant.db
DEFAULT_TIMEZONE=Europe/Amsterdam
```

## Деплой на Railway через GitHub

### 1. Залей проект в GitHub
В репозиторий должны попасть:
- весь код проекта
- `requirements.txt`
- `railway.json`
- `.env.example`
- `.gitignore`

В репозиторий не должны попасть:
- `.env`
- локальная база `.db`
- папка `__pycache__`

### 2. Создай проект в Railway
- New Project
- Deploy from GitHub repo
- выбери этот репозиторий

### 3. Добавь переменные в Railway
В сервисе открой Variables и добавь:

- `BOT_TOKEN`
- `DB_PATH=/data/meeting_assistant.db`
- `DEFAULT_TIMEZONE=Europe/Amsterdam`

### 4. Подключи Volume
Так как бот использует SQLite, для сохранения базы нужен persistent volume.

В Railway:
- открой сервис бота
- создай Volume
- путь монтирования укажи `/data`

После этого файл базы будет жить по пути:

```text
/data/meeting_assistant.db
```

### 5. Перезапусти деплой
Railway возьмёт `startCommand` из `railway.json`:

```bash
python run.py
```

## Важное замечание
Если не подключить Volume, SQLite-база на Railway будет теряться при пересборке или перезапуске контейнера.

## Формат поручения
Команда `/action` ожидает формат:

```text
Исполнитель | Что сделать | Срок
```

Пример:

```text
Иван | Подготовить финмодель | 30.03.2026 18:00
```

## Структура проекта

```text
meeting_assistant_bot/
├── app/
│   ├── bot.py
│   ├── config.py
│   ├── db.py
│   ├── handlers/
│   ├── keyboards/
│   ├── repositories/
│   ├── services/
│   ├── states/
│   └── utils/
├── data/
├── uploads/
├── .env.example
├── .gitignore
├── railway.json
├── requirements.txt
└── run.py
```
