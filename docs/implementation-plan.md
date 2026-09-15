# План реализации «Твой дом»

Основание: техническое задание пользователя от 16 сентября 2026.
Архитектура: React + TypeScript → REST → модульный Go-монолит → PostgreSQL.
Без авторизации, LLM и внешних вызовов. Общий демо-пользователь.

- [x] Проверить правила классификации и переходов тестами, реализовать domain и service.
- [x] Реализовать PostgreSQL repository: атомарное создание с историей, блокировка при смене статуса, сохранение фотографии. Встроенная идемпотентная схема и тестовые справочники.
- [x] Добавить REST controller: JSON и multipart, валидация, 400/404/409/413/500, healthcheck.
- [x] Реализовать React-страницы: главная, форма трёх типов, список, карточка с историей; загрузка, ошибки, пустые состояния.
- [x] Добавить Docker Compose с healthchecks и persistent volume, README и env example.
- [x] Проверить unit tests, сборку TypeScript, реальный API с PostgreSQL, сохранность после рестарта и браузерный сценарий.

Контракты: RequestService.Create(ctx, CreateInput), List, Get, History, Next;
интерфейсы Classifier, Router, RequestRepository, MaxClient, HousingSystemGateway.
POST /api/requests принимает description, address, kind и необязательную photo (multipart).
JSON-вариант принимает те же текстовые поля. kind: PROBLEM, APPLICATION, QUESTION.
Статусы CREATED → SENT → ACCEPTED → IN_PROGRESS → RESOLVED. REJECTED — терминальная альтернативная ветка демо.
Фото JPEG/PNG/WebP до 5 MiB, хранится bytea, выдаётся отдельным endpoint.
Срок — демонстрационные 3 календарных дня, не нормативный срок ЖКХ.

Проверено: Docker Compose build/start всех трёх сервисов; Go tests/vet; TypeScript/Vite build; REST smoke на PostgreSQL; 3 Playwright-сценария; мобильные скриншоты; повторный запуск базы и backend с сохранением данных.
