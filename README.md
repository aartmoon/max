# Твой дом

Рабочий MVP-каркас для хакатона «Умный город»: житель описывает проблему ЖКХ, прикладывает фото, указывает адрес и получает обращение с категорией, ответственной организацией, сроком и историей статусов. Также доступны заявки в УК и вопросы — через ту же форму с полем `kind`.

Это демонстрационный стенд: один общий тестовый пользователь, без авторизации, реального AI и отправки в MAX/ЖКХ. Все открывшие стенд видят одни и те же обращения.

## Быстрый запуск

Нужны Docker Desktop либо Docker Engine с Compose v2 и доступ к интернету для первой сборки.

```bash
docker compose up --build
```

Открыть **http://localhost:3000**. Схема БД и тестовые справочники создаются backend автоматически после готовности PostgreSQL. При перезапуске backend ожидает базу до 30 секунд. Копировать `.env` для первого запуска не обязательно: есть демо-значения по умолчанию.

Для изменения параметров:

```bash
cp .env.example .env
docker compose up --build
```

Фоновый запуск: `docker compose up --build -d`. Проверка состояния: `docker compose ps`. Логи: `docker compose logs -f backend`. Остановка: `docker compose down` — данные сохраняются в томе `postgres_data`. Команда `docker compose down -v` удаляет все демо-данные, включая фотографии.

## Архитектура

```text
React + TypeScript (mobile-first, MAX-ready web app)
              │ REST /api, same origin
       nginx (frontend container)
              │
 Go modular monolith (backend container)
 controller → service → repository → PostgreSQL
                 │
           integration interfaces
           ├── MaxClient → MockMaxClient
           └── HousingSystemGateway → MockHousingSystemGateway
```

Три сервиса Compose: `frontend`, `backend`, `postgres`. nginx раздаёт собранную SPA, перенаправляет `/api` в backend и поддерживает прямое открытие вложенных страниц. CORS для этого не требуется.

Backend: стандартный `net/http`, pgx, явная передача зависимостей в `cmd/server`. `RequestService` координирует сценарий; `Classifier` и `Router` заменяются независимо. `NotificationService` использует `MaxClient`. Создание обращения и начальной истории атомарно; смена статуса использует транзакцию и блокировку строки.

## Структура репозитория

```text
backend/
  cmd/server/main.go       # запуск, сборка зависимостей, graceful shutdown
  config/config.go        # переменные окружения
  controller/http.go      # REST, валидация формата, HTTP-ошибки
  domain/models.go        # User, House, Organization, Request, RequestStatusHistory
  service/
    request.go            # RequestService, NotificationService, repository interface
    rules.go              # Classification / Routing rules, status transitions
    *_test.go             # правила и валидация
  repository/
    postgres.go           # PostgreSQL adapter, транзакции
    schema.sql            # начальная схема и тестовые справочники
  integration/clients.go  # MaxClient, HousingSystemGateway, mock-адаптеры
  Dockerfile
  go.mod / go.sum
frontend/
  src/
    main.tsx              # роуты и оболочка приложения
    api.ts / types.ts     # REST-клиент, типы, подписи
    pages/                # Home, NewRequest, Requests, RequestDetail
    components/UI.tsx     # состояния загрузки, ошибки, статус
    integration/max.ts    # интерфейс MaxBridge и mock
    styles.css
  tests/                  # браузерные проверки Playwright
  nginx.conf / Dockerfile
  package.json / pnpm-lock.yaml / pnpm-workspace.yaml
scripts/smoke.py           # проверка REST на реальной БД
docs/implementation-plan.md
.env.example
docker-compose.yml
```

## Порты и окружение

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `FRONTEND_PORT` | `3000` | Порт приложения на хосте |
| `BACKEND_PORT` | `8080` | Прямой REST-доступ на localhost |
| `POSTGRES_PORT` | `5433` | PostgreSQL на localhost; внутри Compose — 5432 |
| `POSTGRES_USER` | `tvoydom` | Пользователь БД |
| `POSTGRES_PASSWORD` | `tvoydom_demo` | Пароль демонстрационной БД |
| `POSTGRES_DB` | `tvoydom` | Имя БД |
| `MOCK_STATUS_ENABLED` | `true` | REST и UI демо-переходов; `false` отключает |
| `DATABASE_URL` | См. `.env.example` | Для локального Go; Compose формирует внутренний URL сам |
| `PORT` | `8080` | Порт Go при локальном запуске; внутри Compose фиксирован 8080 |

В URL подключения спецсимволы пароля требуют URL-кодирования. Для демо используйте буквенно-цифровой пароль. PostgreSQL применяет начальные логин/пароль только при создании пустого тома; изменение `.env` не меняет существующую роль БД.

## Пользовательский сценарий

1. Открыть главную страницу `/`.
2. Выбрать «Сообщить о проблеме», «Заявка в УК» или «Задать вопрос».
3. На `/requests/new` заполнить описание (5–5000 символов), указать адрес (5–300 символов) либо оставить тестовый: `г. Москва, ул. Тестовая, д. 1`.
4. При желании приложить JPEG, PNG или WebP до 5 МиБ. Фото проверяется по содержимому и сохраняется в `bytea` PostgreSQL; хранилище файлов для MVP не требуется.
5. Нажать «Продолжить»: форма показывает загрузку, ошибки сохраняют введённые данные.
6. На `/requests/:id` увидеть категорию, ответственного, статус, срок, текст обращения, фото и историю.
7. В демо нажимать «Следующий статус» или «Отклонить».
8. В `/requests` открыть сохранённые обращения. Перезагрузка страницы и перезапуск контейнеров данные не удаляют.

Начальный статус — **CREATED / Создано**. Демо-цепочка:

```text
CREATED → SENT → ACCEPTED → IN_PROGRESS → RESOLVED
```

`REJECTED` доступен отдельной демо-кнопкой из любого незавершённого статуса. `RESOLVED` и `REJECTED` терминальные; дальнейший переход возвращает 409.

Срок — **3 календарных дня от создания**, исключительно демонстрационный, не нормативный срок ЖКХ.

## Mock-логика

Правила проверяются по порядку, без учёта регистра, `ё` приравнивается к `е`:

| Подстроки описания | Категория | Ответственный |
| --- | --- | --- |
| `труба`, `течёт` / `течет`, `протечка` | `PIPE_LEAK` | УК «Тестовая» |
| `лифт` | `ELEVATOR` | УК «Тестовая» / подрядчик «ТестЛифт» |
| `отопление`, `батарея`, `холодно` | `HEATING` | УК «Тестовая» |
| Всё остальное | `OTHER` | УК «Тестовая» |

Это поиск подстрок, не морфологический анализ. Текст обращения формируется шаблоном. Произвольный адрес сохраняется как дом, но маршрутизация пока не использует реальные реестры.

`MockMaxClient` записывает событие уведомления в лог. `MockHousingSystemGateway` имитирует отправку при статусе `SENT`. `MockMaxBridge` не делает внешних запросов и позволяет пользоваться приложением в обычном браузере. Никаких токенов MAX сейчас не нужно.

## REST API

| Метод | Путь | Результат |
| --- | --- | --- |
| POST | `/api/requests` | 201: созданное обращение |
| GET | `/api/requests` | Массив обращений общего демо-пользователя |
| GET | `/api/requests/{id}` | Карточка |
| GET | `/api/requests/{id}/history` | История от старого статуса к новому |
| GET | `/api/requests/{id}/photo` | Изображение или 404 |
| POST | `/api/requests/{id}/mock-next-status` | Следующий статус |
| POST | `/api/requests/{id}/mock-next-status?reject=true` | Отклонение |
| GET | `/api/health` | Готовность сервера и БД |
| GET | `/api/config` | Доступность демо-переходов для UI |

JSON без фото:

```bash
curl -X POST http://localhost:3000/api/requests \
  -H 'Content-Type: application/json' \
  -d '{"description":"В подъезде течёт труба","address":"г. Москва, ул. Тестовая, д. 1","kind":"PROBLEM"}'
```

С фото:

```bash
curl http://localhost:3000/api/requests \
  -F 'description=Не работает лифт' \
  -F 'address=г. Москва, ул. Тестовая, д. 1' \
  -F 'kind=PROBLEM' \
  -F 'photo=@/path/to/photo.jpg'
```

`kind`: `PROBLEM` (по умолчанию), `APPLICATION`, `QUESTION`. Ответ содержит обязательные поля Request, а также `address`, `responsibleOrganization`, `kind`, `text`, `hasPhoto`. Даты — ISO 8601, идентификаторы — строки. `userId` и ответственный назначаются сервером.

Ошибки API: JSON `{"error":"сообщение"}`; 400 — валидация/некорректный ID, 404 — нет обращения или фото, 409 — терминальный статус, 413 — HTTP-запрос больше 6 МиБ, 415 — неподдерживаемый Content-Type, 500 — внутренняя ошибка. nginx также ограничивает тело запроса 6 МиБ.

## Разработка и тесты

Локально нужны Go 1.24+ и Node.js 22 с pnpm 11.19.0. Рекомендуемые версии для воспроизводимой сборки заданы Dockerfile и lock-файлами.

```bash
docker compose up -d postgres
# В другом терминале:
cd backend
export DATABASE_URL='postgres://tvoydom:tvoydom_demo@localhost:5433/tvoydom?sslmode=disable'
go run ./cmd/server
# В третьем терминале:
cd frontend
pnpm install --frozen-lockfile
pnpm dev
```

Локальный Vite открывается на 5173 и проксирует `/api` на localhost:8080.

Проверки:

```bash
cd backend
go test ./...
go vet ./...
cd ../frontend
pnpm build
# При запущенном Compose:
pnpm exec playwright install chromium
pnpm test:e2e
cd ..
python3 scripts/smoke.py
```

Тесты создают отдельные демо-обращения в запущенной БД. Удаление не выполняется автоматически. `API_URL` меняет адрес smoke-теста; `BASE_URL` — Playwright.

## Подключение MAX и реальных ЖКХ-систем позже

1. Разместить приложение на публичном HTTPS-адресе и настроить открытие mini app в MAX согласно [официальной документации](https://dev.max.ru/docs/webapps/introduction). Этот репозиторий не регистрирует приложение в MAX и не публикует его автоматически.
2. Реализовать `MaxBridge` вместо `MockMaxBridge`, подключить [официальный MAX Bridge](https://dev.max.ru/docs/webapps/bridge) и его `ready()`; SDK сейчас намеренно не загружается. Учесть back button и параметры запуска.
3. Реализовать backend `MaxClient` для уведомлений, передать его при сборке зависимостей. Токены держать только на сервере. Перед использованием реального пользователя добавить проверку подписанных параметров MAX и заменить `DemoUserID`; авторизация выходит за рамки текущего этапа.
4. Реализовать `HousingSystemGateway` для конкретной системы и подставить в `RequestService`. Сейчас вызов синхронный и mock всегда успешен. Для реальных отправок потребуется надёжная доставка, повторные попытки и сопоставление внешних статусов; нельзя считать лог mock подтверждением доставки. Это можно сделать в монолите с PostgreSQL, без новой инфраструктуры.
5. Заменить `RuleRouter` адаптером справочника домов и организаций, `RuleClassifier` — другой реализацией `Classifier`, когда появится потребность. Текстовый шаблон и правила срока находятся в `service/request.go`.
6. При изменении схемы перейти от начального идемпотентного SQL к последовательным версиям миграций. Текущий `schema.sql` предназначен для первой версии, а не автоматического обновления произвольной существующей схемы.

В MVP нет микросервисов, Kafka, Redis, Kubernetes, Elasticsearch или LLM. Проверка внутри реального MAX требует регистрации приложения и HTTPS; браузерный демо-режим работает без них.

## Agent skills

This repository vendors the [Superpowers](https://github.com/obra/superpowers)
skill set in `.agents/skills`. Codex discovers these skills from the project
automatically. The pinned upstream version and commit are recorded in
`.agents/SUPERPOWERS-SOURCE.md`.
