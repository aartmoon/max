# Arbat Responsibility Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Реализовать безопасную маршрутизацию заявок по домам Арбата, подтверждение результата жителем, переписку с защищёнными вложениями, исполнителей и визиты, а также административное управление организациями и правилами.

**Architecture:** Существующий Go modular monolith сохраняет границы controller → service → repository. Новые небольшие сервисы используют единый policy-компонент доступа, а PostgreSQL-транзакции атомарно меняют статус и аудит; React получает раздельные resident/organization DTO и компактные компоненты карточки.

**Tech Stack:** Go 1.24, net/http, pgx, PostgreSQL, React 19, TypeScript 5.8, Vite, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-29-arbat-responsibility-workflow-design.md`

## Global Constraints

- Сохранять controller → service → repository; бизнес-правила и авторизация находятся на backend.
- Не удалять существующие данные и не создавать параллельную модель адресов или файлов результатов.
- Все даты сохранять в UTC; ошибки API возвращать по-русски.
- Не доверять входным идентификаторам организации, автора или исполнителя.
- Подрядчик не заменяет основного адресата перед жителем.
- Новые фото хранятся только как вложения сообщений; старое `requests.photo` остаётся совместимым.
- MAX, ГАР и ГИС ЖКХ продолжают работать без изменения существующих контрактов.
- Арбатские правила имеют `is_demo=true`, URL источника и дату снимка.

## File map

- `backend/domain/models.go` — новые доменные типы.
- `backend/repository/migrations/009_request_workflow.sql` — схема и перенос старых заявок.
- `backend/repository/migrations/010_arbat_responsibility_snapshot.sql` — организации и демо-правила Арбата.
- `backend/repository/{routing,request_access,messages,workflow}.go` — сфокусированные PostgreSQL-адаптеры.
- `backend/service/{routing,access,messages,workflow,routing_admin}.go` — бизнес-правила.
- `backend/controller/{messages,workflow,routing_admin}.go` — HTTP endpoints.
- `frontend/src/components/{RequestConversation,RequestAssignment,ResidentResolution,RoutingAdmin}.tsx` — новые блоки UI.
- `frontend/src/pages/{RequestDetail,Admin,NewRequest}.tsx` — композиция экранов.

---

### Task 1: Safe schema migration and Arbat snapshot

**Files:**
- Create: `backend/repository/migrations/009_request_workflow.sql`
- Create: `backend/repository/migrations/010_arbat_responsibility_snapshot.sql`
- Modify: `backend/repository/postgres.go`
- Modify: `backend/domain/models.go`
- Test: `backend/repository/workflow_migration_integration_test.go`

**Interfaces:** Produces `Organization`, `ResponsibilityRule`, `RequestMessage`, `MessageAttachment`, `ResolutionFeedback` and schema versions 9–10.

- [ ] **Step 1: Write the failing migration integration test**

Guard it with `TEST_DATABASE_URL`. Insert a legacy request before versions 9–10, run `Migrate`, and assert the row remains, status stays `RESOLVED`, and `primary_organization_name_snapshot` equals its former organization. Assert at least three demo rules, a `RESOURCE_SUPPLIER`, a `HOUSING_INSPECTION`, non-empty source URLs, and idempotent re-run.

```go
require.Equal(t, "RESOLVED", status)
require.Equal(t, "УК «Тестовая»", snapshot)
require.GreaterOrEqual(t, demoRules, 3)
```

- [ ] **Step 2: Run RED**

Run: `cd backend && TEST_DATABASE_URL="$DATABASE_URL" go test ./repository -run TestWorkflowMigrationsPreserveLegacyRequestAndSeedArbat -count=1`

Expected: FAIL because the new tables/columns do not exist.

- [ ] **Step 3: Add domain types and migration 009**

Add exact structs with JSON tags: extended `Organization`; `ResponsibilityRule` with house/category/place/urgency/role/validity/source/demo; `RequestMessage` with author role and attachments. Migration 009 creates eight organization types, extends organizations, creates rules, adds `ROUTING_REQUIRED` and `CLOSED`, adds route snapshots/assignment/visit/report/awaiting/reopen fields, creates feedback/messages/attachments, and backfills every old responsible organization snapshot without deleting data.

- [ ] **Step 4: Add idempotent migration 010**

Use external GUID/OGRN/INN conflict keys. Seed real organizations returned by public Арбат house profiles, including house managers, АО «МОСВОДОКАНАЛ», АО «МОСЭНЕРГОСБЫТ», ПАО «МОЭК», the municipal organization and Мосжилинспекция. Join existing houses through GAR GUIDs; never create address rows. All seeded rules use `is_demo=true` with source URL/date.

- [ ] **Step 5: Register migrations and run GREEN**

Embed both files in `postgres.go`, apply in version order under the advisory lock, then run the focused test and `cd backend && go test ./repository -count=1`.

- [ ] **Step 6: Commit**

```bash
git add backend/domain/models.go backend/repository/postgres.go backend/repository/migrations/009_request_workflow.sql backend/repository/migrations/010_arbat_responsibility_snapshot.sql backend/repository/workflow_migration_integration_test.go
git commit -m "feat: add responsibility workflow schema"
```

---

### Task 2: Rule-based routing and safe creation

**Files:**
- Create: `backend/service/routing.go`
- Create: `backend/repository/routing.go`
- Modify: `backend/service/request.go`
- Modify: `backend/repository/postgres.go`
- Test: `backend/service/routing_test.go`
- Test: `backend/service/request_test.go`
- Test: `backend/repository/routing_integration_test.go`

**Interfaces:**

```go
type RouteQuery struct { HouseID, Category, Place, Urgency string; At time.Time }
type RouteDecision struct { Primary, Contractor, Escalation *domain.ResponsibilityRule; Reason string }
type ResponsibilityRepository interface { FindActiveRules(context.Context, RouteQuery) ([]domain.ResponsibilityRule, error) }
func (s RoutingService) Resolve(context.Context, RouteQuery) (RouteDecision, error)
func DetectImmediateDanger(string) bool
```

- [ ] **Step 1: Write failing tests**

Cover exact-over-wildcard precedence, separate contractor, expired/inactive exclusion, missing and equal-best primary routes, and danger phrases for fire/gas/life threat. Add create tests proving no rule yields `ROUTING_REQUIRED` instead of a random organization.

- [ ] **Step 2: Run RED**

Run: `cd backend && go test ./service -run 'TestResolve|TestDetectImmediateDanger|TestCreateRoutes' -count=1`.

- [ ] **Step 3: Implement routing**

Match category exactly. Non-empty place/urgency fields must equal query; exact optional fields increase specificity. Reject equal-best primary ambiguity as manual routing. Normalize danger text for case and `ё` and return a Russian 112 instruction before persistence.

- [ ] **Step 4: Integrate creation**

Extend `CreateInput` with place/urgency. Resolve existing house first, then rules. Save primary/contractor and immutable names/reason/source. On no primary save `ROUTING_REQUIRED`. Insert request, initial history and initial `SYSTEM_EVENT` in one transaction. Accept no organization ID from input.

- [ ] **Step 5: Run GREEN and commit**

```bash
cd backend && go test ./service -count=1
go test ./repository -run 'TestFindActiveRules|TestArbatRoutes' -count=1
git add backend/service/routing.go backend/service/routing_test.go backend/service/request.go backend/service/request_test.go backend/repository/routing.go backend/repository/routing_integration_test.go backend/repository/postgres.go
git commit -m "feat: route requests by house responsibility"
```

---

### Task 3: Central access policy and organization queues

**Files:**
- Create: `backend/service/access.go`
- Create: `backend/repository/request_access.go`
- Modify: `backend/controller/http.go`
- Modify: `backend/controller/admin.go`
- Test: `backend/service/access_test.go`
- Test: `backend/controller/roles_test.go`
- Test: `backend/repository/request_access_integration_test.go`

**Interfaces:** `AccessService.Require(ctx, requestID, capability)` and server-side `RequestFilters{Queue, UserID, VisitDayStartUTC, VisitDayEndUTC}`.

- [ ] **Step 1: Write failing access tests**

Cover resident owner/non-owner, primary manager, contractor manager, current assignee, foreign manager and admin. Direct detail/history/legacy-photo/attachment URLs for a foreign manager must all return 403.

- [ ] **Step 2: Run RED**

Run: `cd backend && go test ./service ./controller -run 'Access|OtherOrganization|Photo' -count=1` and confirm the current empty-scope admin photo path fails the expected assertion.

- [ ] **Step 3: Implement access SQL and policy**

Join owner, primary organization, contractor, assignee and assignee organization. Admin bypass is role-only. Managers require server-side `OrganizationID`. Capabilities: `read`, `write_public`, `write_internal`, `manage_work`, `admin`.

- [ ] **Step 4: Replace empty-scope reads and add queues**

Apply the policy to card, history, legacy photo and future attachments. Server-filter organization lists for `ACTIVE`, `UNASSIGNED`, `MINE`, `VISIT_TODAY`, `ALL`; `MINE` uses session user ID.

- [ ] **Step 5: Run GREEN and commit**

```bash
cd backend && go test ./service ./controller ./repository -run 'Access|Organization|Photo|Queue' -count=1
git add backend/service/access.go backend/service/access_test.go backend/repository/request_access.go backend/repository/request_access_integration_test.go backend/controller/http.go backend/controller/admin.go backend/controller/roles_test.go
git commit -m "fix: enforce request organization isolation"
```

---

### Task 4: Messages and protected attachments

**Files:**
- Create: `backend/service/messages.go`
- Create: `backend/repository/messages.go`
- Create: `backend/controller/messages.go`
- Test: `backend/service/messages_test.go`
- Test: `backend/repository/messages_integration_test.go`
- Modify: `backend/controller/http.go`

**Interfaces:**

```go
const MaxMessageAttachments = 5
const MaxMessageAttachmentSize = 5 << 20
const MaxMessageAttachmentsTotal = 20 << 20
type NewAttachment struct { Name string; Data []byte }
type NewMessageInput struct { RequestID, Type, Text string; Attachments []NewAttachment }
func (s MessageService) List(context.Context, string) ([]domain.RequestMessage, error)
func (s MessageService) Create(context.Context, NewMessageInput) (domain.RequestMessage, error)
```

- [ ] **Step 1: Write failing tests**

Prove residents cannot create internal/organization messages; foreign managers cannot post; resident lists exclude internal notes; request-info sets awaiting `RESIDENT`; resident reply sets awaiting `ORGANIZATION` without changing status; content-sniffed JPEG/PNG/WebP pass while spoofed files fail.

- [ ] **Step 2: Run RED**

Run: `cd backend && go test ./service -run 'Message|Attachment|InformationRequest' -count=1`.

- [ ] **Step 3: Implement service and repository**

Require text or attachment, cap text at 5000 runes and files at 5/5 MiB/20 MiB. Derive author/name/role from current user. Normalize filename with `path.Base` and control-character removal. Persist message, files and awaiting party atomically. Filter internal types in resident SQL.

- [ ] **Step 4: Add endpoints and resilient notification**

Add resident and organization `GET/POST .../{id}/messages` plus `GET /api/request-attachments/{id}`. Attachment response uses stored detected MIME, `nosniff`, `no-store`. Notify only after commit; a notifier error must leave the saved message intact.

- [ ] **Step 5: Run GREEN and commit**

```bash
cd backend && go test ./service ./repository ./controller -run 'Message|Attachment|InformationRequest' -count=1
git add backend/service/messages.go backend/service/messages_test.go backend/repository/messages.go backend/repository/messages_integration_test.go backend/controller/messages.go backend/controller/http.go
git commit -m "feat: add protected request conversations"
```

---

### Task 5: Assignment, visits and resident closure

**Files:**
- Create: `backend/service/workflow.go`
- Create: `backend/repository/workflow.go`
- Create: `backend/controller/workflow.go`
- Test: `backend/service/workflow_test.go`
- Test: `backend/repository/workflow_integration_test.go`
- Modify: `backend/service/admin.go`
- Modify: `backend/controller/admin.go`

**Interfaces:**

```go
type AssignmentInput struct { AssigneeUserID, ContractorOrganizationID, ExecutorContact string; VisitStart, VisitEnd *time.Time }
type ResolveInput struct { FinalReport string }
type ResidentDecisionInput struct { Solved bool; Rating *int; Comment string }
```

- [ ] **Step 1: Write failing tests**

Cover legal status transitions, required final report, manager cannot close, resident owns the `RESOLVED` request, solved→`CLOSED`, unsolved comment requirement and reopen counter, terminal `CLOSED`, eligible assignee organizations, visit start≤end and no past start.

- [ ] **Step 2: Run RED**

Run: `cd backend && go test ./service -run 'Workflow|Assignment|Visit|Resolution|Closed' -count=1`.

- [ ] **Step 3: Implement validation and atomic persistence**

Inject `Now` for UTC validation. Lock request `FOR UPDATE`. Write status, feedback/reopen count and history atomically. Assignment/contractor/visit changes insert `SYSTEM_EVENT` with the authenticated real user. Reject every work mutation after `CLOSED`.

- [ ] **Step 4: Add endpoints**

Add organization assignment/status endpoints, resident resolution endpoint and admin manual-route endpoint. Actor, organization and assignee membership come from session/server records only.

- [ ] **Step 5: Run GREEN and commit**

```bash
cd backend && go test ./service ./repository ./controller -run 'Workflow|Assignment|Visit|Resolution|Closed' -count=1
git add backend/service/workflow.go backend/service/workflow_test.go backend/repository/workflow.go backend/repository/workflow_integration_test.go backend/controller/workflow.go backend/service/admin.go backend/controller/admin.go
git commit -m "feat: add assignment and resident closure workflow"
```

---

### Task 6: Admin organizations, rules and access API

**Files:**
- Create: `backend/service/routing_admin.go`
- Create: `backend/controller/routing_admin.go`
- Test: `backend/service/routing_admin_test.go`
- Modify: `backend/repository/routing.go`
- Modify: `backend/repository/auth.go`
- Modify: `backend/controller/admin.go`

**Interfaces:** `SaveOrganization(ctx, optionalID, input)` and `SaveRule(ctx, optionalID, input)` with explicit fields from the design.

- [ ] **Step 1: Write failing validation/access tests**

Test admin-only mutation, known organization type/house/org, date order, valid rule role, source required for demo rules, deactivation instead of deletion, and access assignment only to an organization-capable user.

- [ ] **Step 2: Run RED**

Run: `cd backend && go test ./service ./controller -run 'RoutingAdmin|ResponsibilityRule|OrganizationAccess' -count=1`.

- [ ] **Step 3: Implement CRUD and conflicts**

Return conflict for equal-specificity active primary overlap in intersecting validity periods. Do not expose destructive delete for referenced records. Managers get read-only own-organization data; admins get all.

- [ ] **Step 4: Add API and run GREEN**

Add admin GET/POST/PATCH organizations and responsibility-rules endpoints. Keep existing user role/organization endpoints admin-only.

- [ ] **Step 5: Commit**

```bash
git add backend/service/routing_admin.go backend/service/routing_admin_test.go backend/repository/routing.go backend/repository/auth.go backend/controller/routing_admin.go backend/controller/admin.go
git commit -m "feat: manage responsibility rules and access"
```

---

### Task 7: Frontend contracts, protected routes and queues

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/main.tsx`
- Modify: `frontend/src/admin.ts`
- Modify: `frontend/src/pages/Admin.tsx`
- Modify: `frontend/src/pages/NewRequest.tsx`
- Test: `frontend/tests/admin.spec.ts`
- Test: `frontend/tests/requests.spec.ts`

- [ ] **Step 1: Write failing Playwright tests**

Mock resident/manager/admin `/api/me`. Resident sees no organization link and direct `/admin` redirects home. Manager sees only server-returned rows. New queue controls send `UNASSIGNED`, `MINE`, `VISIT_TODAY` query parameters.

- [ ] **Step 2: Run RED**

Run: `cd frontend && pnpm test:e2e --grep 'доступ|Без исполнителя|Мои заявки|Визит сегодня'`.

- [ ] **Step 3: Update contracts and routing**

Add statuses, routing snapshots, nullable organization, contractor, assignee, visit, result, awaiting and reopen fields. Add resident/organization/admin API clients. Add `RequireOrganizationRole`; it redirects residents without rendering organization pages, while backend remains authoritative.

- [ ] **Step 4: Update filters and creation form**

Only `CLOSED` and `REJECTED` are terminal. Add the three queues. Add place and urgency controls, danger guidance, retry-preserving errors and duplicate-submit blocking to new request.

- [ ] **Step 5: Run GREEN and commit**

```bash
cd frontend && pnpm build
pnpm test:e2e --grep 'доступ|очеред|создание заявки'
git add frontend/src/types.ts frontend/src/api.ts frontend/src/main.tsx frontend/src/admin.ts frontend/src/pages/Admin.tsx frontend/src/pages/NewRequest.tsx frontend/tests/admin.spec.ts frontend/tests/requests.spec.ts
git commit -m "feat: add protected organization queues"
```

---

### Task 8: Request-card collaboration UI

**Files:**
- Create: `frontend/src/components/RequestConversation.tsx`
- Create: `frontend/src/components/RequestAssignment.tsx`
- Create: `frontend/src/components/ResidentResolution.tsx`
- Create: `frontend/src/components/RoutingAdmin.tsx`
- Modify: `frontend/src/pages/RequestDetail.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/tests/request-workflow.spec.ts`

- [ ] **Step 1: Write failing scenarios**

Test recipient/reason/contractor/visit display; no resident internal note; two-image public message without status change; failed send preserves form and retry succeeds; manager internal note and info request; report required for resolve; resident close with rating; resident reopen requires comment; admin manual routes and edits demo rule.

- [ ] **Step 2: Run RED**

Run: `cd frontend && pnpm test:e2e tests/request-workflow.spec.ts`.

- [ ] **Step 3: Build focused components**

Each component owns loading/error/busy state and accepts `onChanged(): Promise<void>`. Conversation renders author/role/time/type and protected attachment links. Assignment validates local input. Resolution appears only for `RESOLVED`. RoutingAdmin appears only to admin.

- [ ] **Step 4: Compose detail and mobile styles**

Load request/history/messages with specific retry controls. Render empty conversation/assignee/visit states and result separately. Preserve secured legacy photo. Keep one-column mobile layout, long-name wrapping and 44px touch targets.

- [ ] **Step 5: Run GREEN and commit**

```bash
cd frontend && pnpm build && pnpm test:e2e tests/request-workflow.spec.ts
git add frontend/src/components/RequestConversation.tsx frontend/src/components/RequestAssignment.tsx frontend/src/components/ResidentResolution.tsx frontend/src/components/RoutingAdmin.tsx frontend/src/pages/RequestDetail.tsx frontend/src/styles.css frontend/tests/request-workflow.spec.ts
git commit -m "feat: add request collaboration interfaces"
```

---

### Task 9: Regression, documentation and acceptance verification

**Files:**
- Modify: `README.md`
- Modify: `scripts/smoke.py`
- Modify: focused implementation files only for observed failures

- [ ] **Step 1: Extend real-DB smoke checks**

Authenticate fixture resident/manager/admin sessions; create all three route categories; verify manual fallback; foreign-manager 403 for detail/photo/attachment; public/internal messages; visit; resolve→reopen→resolve→close.

- [ ] **Step 2: Run smoke and fix observed failures**

Run `docker compose up --build -d` then `python3 scripts/smoke.py`. Expected: exit 0 and explicit output for every acceptance case.

- [ ] **Step 3: Update README**

Document rule precedence, organization roles, statuses, message visibility, visit UTC rules, admin management, Arbat snapshot provenance/demo warning, API paths, MIME/size limits and all verification commands.

- [ ] **Step 4: Format**

Run `cd backend && gofmt -w domain controller service repository cmd integration`. For frontend use an already-installed formatter only; otherwise preserve current formatting and rely on TypeScript build.

- [ ] **Step 5: Run fresh backend verification**

Run `cd backend && go test ./... -count=1` and `cd ../gar-init && go test ./... -count=1`. Require zero failures.

- [ ] **Step 6: Run fresh frontend verification**

Run `cd frontend && pnpm build && pnpm test:e2e`. Require exit 0 and all scenarios passing.

- [ ] **Step 7: Run PostgreSQL integration verification**

Run `cd backend && TEST_DATABASE_URL='postgres://tvoydom:tvoydom_demo@localhost:5433/tvoydom?sslmode=disable' go test ./repository -count=1`.

- [ ] **Step 8: Audit requirements and diff**

Run `git diff --check` and `git status --short`, inspect every changed file, and map each acceptance criterion to a passing test. Report any unavailable check with its exact blocker.

- [ ] **Step 9: Commit**

```bash
git add README.md scripts/smoke.py backend frontend
git commit -m "docs: document responsibility workflow"
```
