# Meeting Assistant Telegram Bot (Go)

Telegram-бот на Go для ведения встреч из нескольких источников:

- Zoom
- Яндекс Телемост
- загруженное аудио/voice
- ручная фиксация оффлайн-встречи

## Что уже умеет MVP

- создавать сущность встречи
- выбирать источник: `zoom | telemost | offline | upload`
- добавлять участников
- вести ручные заметки во время встречи: `/note`, `/decision`, `/action`
- загружать аудио/voice после встречи
- делать расшифровку через OpenAI-compatible API, если задан `OPENAI_API_KEY`
- собирать итоговый протокол
- сохранять поручения как задачи, связанные со встречей
- принимать Zoom webhook как основу для auto-flow
- поддерживать Telemost в semi-auto/fallback-режиме


## Быстрый старт

### 1. Подготовка

```bash
cp .env.example .env
```

Заполнить минимум:

- `BOT_TOKEN`
- `DB_PATH`

Для транскрибации и AI-протоколов дополнительно:

- `OPENAI_API_KEY`
- при необходимости `OPENAI_BASE_URL`

### 2. Запуск

```bash
go mod tidy
go run ./cmd/bot
```

### 3. Команды бота

- `/start` — главное меню
- `/new_meeting` — создать встречу
- `/my_meetings` — последние встречи
- `/note текст` — заметка в активную встречу
- `/decision текст` — решение
- `/action ФИО | дедлайн | текст` — поручение
- `/finish` — завершить встречу и собрать протокол
- `/cancel` — сбросить активный wizard

## Основной flow

1. Пользователь нажимает `Создать встречу`
2. Выбирает источник
3. Вводит название
4. По желанию добавляет участников
5. Во время встречи фиксирует `/note`, `/decision`, `/action`
6. После встречи загружает аудио/voice или завершает вручную
7. Бот формирует протокол
8. Поручения попадают в таблицу `tasks`

## Zoom flow

### Что реализовано

- HTTP endpoint `/webhooks/zoom`
- создание/обновление встречи по `source_meeting_id`
- прием простого payload для завершенной встречи и transcript metadata

### Что нужно доделать под прод

- строгую валидацию `x-zm-signature`
- отдельную обработку challenge/event validation
- загрузку transcript/recording по Zoom API access token
- RTMS consumer для near real-time режима

## Telemost flow

### Что реализовано

- отдельный тип встречи `telemost`
- карточка встречи и fallback-режим
- загрузка записи/voice вручную после встречи

### MVP идея

Если Telemost summary/transcript нельзя получить автоматически, организатор просто пересылает запись в бот. После этого пайплайн такой же, как у `upload/offline`.

## Структура проекта

```text
cmd/bot                     entrypoint
internal/app                DI и запуск
internal/bot                Telegram handlers
internal/config             env config
internal/domain             сущности
internal/service            meeting/protocol/stt/task services
internal/storage            sqlite repository
internal/adapter/zoom       webhook adapter
internal/adapter/telemost   placeholder adapter
migrations                  schema.sql
```

## Таблицы

- `meetings`
- `meeting_participants`
- `meeting_artifacts`
- `meeting_items`
- `tasks`

## Ограничения текущей версии

- Zoom и Telemost не доведены до полной боевой OAuth-интеграции
- для STT нужен внешний API
- NLP-извлечение сделано либо через LLM, либо через простой fallback
- нет multi-tenant ACL и ролей
- нет planner API, задачи пока хранятся внутри этой БД
