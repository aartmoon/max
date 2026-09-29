# MAX Bot Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver real MAX notifications to linked residents and organization managers while preserving the existing `/start` flow.

**Architecture:** Validate and persist MAX Mini App identities behind the authenticated API, then route typed request events through a repository-backed notifier into the existing MAX API client. The frontend links the current account once MAX Bridge and application authentication are both available.

**Tech Stack:** Go 1.24, PostgreSQL, React 19, TypeScript, Playwright, MAX Bot API.

**Spec:** `docs/superpowers/specs/2026-09-29-max-bot-notifications-design.md`

## Global Constraints

- Internal notes are never sent to MAX.
- Missing MAX links and delivery errors never roll back business operations.
- Raw `initData` and bot tokens are never logged.
- Existing long polling remains unchanged.

---

### Task 1: Validate and store MAX account links

**Files:**
- Create: `backend/integration/maxbot/launch_data.go`
- Create: `backend/integration/maxbot/launch_data_test.go`
- Create: `backend/repository/migrations/012_max_accounts.sql`
- Create: `backend/repository/max_accounts.go`
- Modify: `backend/repository/postgres.go`

**Interfaces:**
- Produces: `ValidateLaunchData(raw, token string, now time.Time) (Identity, error)` where `Identity` contains `UserID int64` and `ChatID int64`.
- Produces: `LinkMAXAccount(ctx, appUserID string, maxUserID, chatID int64) error`.

- [ ] Write tests that sign realistic launch data and assert valid parsing, invalid signature rejection, duplicate-field rejection, stale `auth_date` rejection, and non-dialog rejection.
- [ ] Run `go test ./integration/maxbot -run LaunchData -count=1` and confirm the new tests fail because the validator is absent.
- [ ] Implement strict URL parsing, sorted HMAC-SHA256 validation, constant-time hash comparison, 10-minute freshness, and JSON identity extraction.
- [ ] Run the focused tests and confirm they pass.
- [ ] Add migration 012 with `user_id` primary key, unique `max_user_id`, `chat_id`, and timestamps; embed and execute it from `Postgres.Migrate`.
- [ ] Add the idempotent upsert repository method and run `go test ./repository -count=1`.

### Task 2: Expose authenticated account linking

**Files:**
- Create: `backend/service/max_account.go`
- Create: `backend/service/max_account_test.go`
- Modify: `backend/controller/http.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Produces: `MAXAccountService.Link(ctx context.Context, appUserID, rawInitData string) error`.
- Produces: authenticated `POST /api/me/max-account` with `{ "initData": "..." }`.

- [ ] Write a failing service test proving only validated identity data reaches the repository and invalid data is rejected.
- [ ] Run `go test ./service -run MAXAccount -count=1` and verify the expected failure.
- [ ] Implement the service with injected token/time and add it to `controller.Handler`.
- [ ] Add a size-limited authenticated endpoint returning `{ "ok": true }`.
- [ ] Run the focused service and controller tests.

### Task 3: Send typed resident and manager notifications

**Files:**
- Create: `backend/service/max_notifications.go`
- Create: `backend/service/max_notifications_test.go`
- Create: `backend/repository/max_notifications.go`
- Modify: `backend/integration/maxbot/client.go`
- Modify: `backend/integration/clients.go`
- Modify: `backend/service/request.go`
- Modify: `backend/controller/messages.go`
- Modify: `backend/controller/workflow.go`
- Modify: `backend/controller/admin.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Produces: `NotifyOwner(ctx, request, event)` and `NotifyManagers(ctx, request, event)`.
- Consumes: repository destinations `{ UserID int64, ChatID int64 }` and MAX `Send`.

- [ ] Write failing tests for owner delivery, primary/contractor manager delivery, deduplication, missing links, internal-note exclusion, Russian status labels, excerpts, and the mini-app deep link `https://max.ru/<username>?startapp=request_<id>`.
- [ ] Run `go test ./service -run MAXNotification -count=1` and verify failures are caused by missing behavior.
- [ ] Add repository queries for the request owner and managers with `manager`/`admin` roles in the primary or contractor organization.
- [ ] Implement best-effort typed notification formatting and sending through the real MAX client.
- [ ] Wire request creation, public messages, statuses, assignment, routing, and resolution events; use the logging sender when the token is absent.
- [ ] Run focused backend tests and keep email notifications unchanged.

### Task 4: Link from the MAX mini app and handle request deep links

**Files:**
- Modify: `frontend/src/api.ts`
- Modify: `frontend/src/integration/max.ts`
- Modify: `frontend/src/integration/MaxIntegration.tsx`
- Modify: `frontend/src/auth.tsx`
- Modify: `frontend/tests/max-bridge.spec.ts`

**Interfaces:**
- Consumes: `authApi.linkMAX(initData)` and `launchRoute("request_<id>", roles)`.

- [ ] Extend the Playwright test to expect one authenticated link request in MAX, no request in browser mode, and safe routing for numeric `request_<id>` payloads.
- [ ] Run `pnpm test:e2e -- max-bridge.spec.ts` and verify the new assertions fail.
- [ ] Add the API method, trigger it once after both auth and bridge are ready, and ignore background link failure so the application stays usable.
- [ ] Route residents to `/requests/<id>` and organization staff to `/admin/requests/<id>`; continue rejecting arbitrary payloads.
- [ ] Run the focused Playwright test.

### Task 5: Configuration, documentation, and verification

**Files:**
- Modify: `.env.example`
- Modify: `README.md`

- [ ] Document automatic MAX linking, recipient rules, and the requirement for `MAX_BOT_TOKEN`, `MAX_APP_URL`, and `MAX_BOT_USERNAME`.
- [ ] Format changed Go files with `gofmt`.
- [ ] Run `go test ./...` from `backend`.
- [ ] Run `pnpm build` and `pnpm test:e2e -- max-bridge.spec.ts` from `frontend`.
- [ ] Run `git diff --check` and review the final diff against the spec.
