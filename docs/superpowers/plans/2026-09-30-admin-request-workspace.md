# Admin Request Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a scalable manager/admin request workspace with server-side search, filtering, summaries, pagination, and a separate administrator settings area.

**Architecture:** Replace the array response from `GET /api/admin/requests` with a typed paginated response produced by one access-scoped repository operation. Keep URL state in `Admin.tsx`, render focused desktop/mobile list components, and move user, organization, and routing configuration to administrator-only settings routes.

**Tech Stack:** Go 1.24, PostgreSQL/pgx, React 19, TypeScript 5.7, React Router 7, Vite 6, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-30-automatic-routing-and-admin-workspace-design.md`

## Global Constraints

- Page sizes are exactly `20`, `50`, or `100`; the default is `20`.
- All access predicates, search, filters, counts, sorting, and pagination execute on the server.
- Managers cannot widen their organization scope with query parameters.
- Request-list state is reproducible from the URL.
- Changing queue, search, filters, sort, or page size resets the page to `1`.
- Existing results remain visible during refresh/page errors.
- `/admin` contains no organization, user, or routing-rule forms.
- `/admin/settings/**` is visible only to administrators; its APIs remain server-protected.
- Desktop uses a table and mobile uses cards without page-level horizontal overflow at 320 px.
- Production code is written only after its focused test fails for the expected reason.

---

### Task 1: Define and validate the paginated request-list contract

**Files:**
- Modify: `backend/domain/models.go`
- Create: `backend/controller/admin_requests.go`
- Create: `backend/controller/admin_requests_test.go`
- Modify: `backend/controller/admin.go`

**Interfaces:**
- Produces: `domain.AdminRequestQuery`, `domain.AdminRequestSummary`, and `domain.AdminRequestPage`.
- Produces: `parseAdminRequestQuery(url.Values, domain.User, time.Time) (domain.AdminRequestQuery, error)`.
- Consumed by: repository work in Task 2 and frontend contract in Task 3.

- [ ] **Step 1: Add failing parser tests**

Cover defaults and a full query:

```go
func TestParseAdminRequestQueryDefaults(t *testing.T) {
	got, err := parseAdminRequestQuery(url.Values{}, domain.User{ID: "7", Roles: []string{"admin"}}, time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC))
	if err != nil { t.Fatal(err) }
	if got.Page != 1 || got.PageSize != 20 || got.Queue != "ACTIVE" || got.Sort != "PRIORITY" || got.UserID != "7" {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}
```

Table-test invalid `page=0`, `page=x`, `pageSize=25`, unsupported queue/status/kind/sort, malformed organization ID, and only one of `visitStart`/`visitEnd`. Assert a `domain.ValidationError` for every case. Test that a manager's `OrganizationID` overrides an `organization=999` parameter while an admin may use `organization=999`.

- [ ] **Step 2: Verify parser tests fail**

Run: `cd backend && go test ./controller -run TestParseAdminRequestQuery -count=1`

Expected: compile failure because the types and parser do not exist.

- [ ] **Step 3: Add exact domain types**

```go
type AdminRequestQuery struct {
	OrganizationID string
	UserID         string
	Query          string
	Queue          string
	Status         string
	Kind           string
	Sort           string
	Page           int
	PageSize       int
	VisitStart     time.Time
	VisitEnd       time.Time
	Now            time.Time
}

type AdminRequestSummary struct {
	Active int `json:"active"`
	New int `json:"new"`
	Overdue int `json:"overdue"`
	Unassigned int `json:"unassigned"`
	Done int `json:"done"`
}

type AdminRequestPage struct {
	Items []Request `json:"items"`
	Page int `json:"page"`
	PageSize int `json:"pageSize"`
	Total int `json:"total"`
	Summary AdminRequestSummary `json:"summary"`
}
```

- [ ] **Step 4: Implement strict query parsing**

Use allow-list maps for queue, status, kind, sort, and page size. Trim `q`; parse positive numeric IDs; set `Now` once; validate the visit interval; and return Russian `domain.ValidationError` messages. Do not silently coerce invalid values.

- [ ] **Step 5: Replace the controller's ad-hoc parameters**

In `GET /api/admin/requests`, call `parseAdminRequestQuery`, return validation failures through `respond`, and call:

```go
page, err := h.Repo.ListAccessibleRequests(r.Context(), query)
respond(w, page, err)
```

- [ ] **Step 6: Run parser/controller tests**

Run: `cd backend && go test ./controller -count=1`

Expected: PASS.

- [ ] **Step 7: Commit the HTTP contract**

```bash
git add backend/domain/models.go backend/controller/admin.go backend/controller/admin_requests.go backend/controller/admin_requests_test.go
git commit -m "feat: define paginated admin request API"
```

### Task 2: Implement access-scoped SQL search, filters, counts, and pagination

**Files:**
- Modify: `backend/repository/request_access.go`
- Create: `backend/repository/admin_requests_integration_test.go`

**Interfaces:**
- Consumes: `domain.AdminRequestQuery` from Task 1.
- Produces: `Postgres.ListAccessibleRequests(context.Context, domain.AdminRequestQuery) (domain.AdminRequestPage, error)`.

- [ ] **Step 1: Add failing integration coverage**

Use an isolated migrated schema. Insert two organizations, an admin, managers in separate organizations, and requests covering `ROUTING_REQUIRED`, `CREATED`, `SENT`, `ACCEPTED`, `IN_PROGRESS`, `CLOSED`, `REJECTED`, emergency, overdue, unassigned, assigned-to-current-user, and visits inside/outside the requested day.

Create subtests named `access`, `search_tokens`, `queues`, `combined_filters`, `priority_sort`, `pagination`, and `summary`. Assert, for example:

```go
page, err := repo.ListAccessibleRequests(ctx, domain.AdminRequestQuery{
	OrganizationID: orgOne, UserID: managerOne, Query: "арбат лифт",
	Queue: "ALL", Sort: "NEWEST", Page: 1, PageSize: 20, Now: now,
})
if err != nil { t.Fatal(err) }
if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Description != "Не работает лифт" {
	t.Fatalf("unexpected search page: %+v", page)
}
```

For pagination, insert 25 accessible requests, request pages 1 and 2 with size 20, assert counts `20` and `5`, stable non-overlapping IDs, and `Total == 25`. Assert a beyond-last page has zero items but retains `Total == 25`.

- [ ] **Step 2: Verify integration tests fail**

Run: `cd backend && WORKFLOW_TEST_DATABASE_URL="$DATABASE_URL" go test ./repository -run TestAdminRequestListing -count=1`

Expected: compile failure because the repository still has the old signature.

- [ ] **Step 3: Build a shared access-scoped filtered query**

Keep SQL values bound. Build only allow-listed predicate fragments in Go. The base access predicate must remain:

```sql
($organization = '' OR
 r.primary_organization_id::text = $organization OR
 r.contractor_organization_id::text = $organization OR
 EXISTS (
   SELECT 1 FROM user_organizations access_uo
   WHERE access_uo.user_id=r.assigned_user_id
     AND access_uo.organization_id::text=$organization
 ))
```

Tokenize the trimmed search query with `strings.Fields`. For each token append a bound `%token%` argument and a grouped `ILIKE` predicate covering request ID, address, description, primary/contractor snapshots, and assignee name.

- [ ] **Step 4: Implement exact queue and sort mappings**

Use fixed queue fragments. Important mappings:

```go
"NEW":       `r.status IN ('ROUTING_REQUIRED','CREATED','SENT')`,
"IN_WORK":   `r.status IN ('ACCEPTED','IN_PROGRESS')`,
"DONE":      `r.status IN ('CLOSED','REJECTED')`,
"MINE":      `r.assigned_user_id::text=$userID AND r.status NOT IN ('CLOSED','REJECTED')`,
"EMERGENCY": `r.kind='EMERGENCY' AND r.status NOT IN ('CLOSED','REJECTED')`,
```

Map sort values to static clauses. `PRIORITY` orders emergency active first, then overdue active, then new statuses, then deadline and creation time. Always end with `r.id DESC` for stable pages.

- [ ] **Step 5: Return rows, total, and stable summary**

Run the item query with `LIMIT` and `OFFSET`, a filtered `count(*)` query for `total`, and a summary query using PostgreSQL `FILTER` over the same access/organization scope while intentionally excluding queue, status, kind, and search. Return `Items: []domain.Request{}` rather than `nil`.

- [ ] **Step 6: Run repository and backend suites**

Run: `cd backend && go test ./repository -run TestAdminRequestListing -count=1`

Then: `cd backend && go test ./... -count=1`

Expected: PASS.

- [ ] **Step 7: Commit server-side listing**

```bash
git add backend/repository/request_access.go backend/repository/admin_requests_integration_test.go
git commit -m "feat: paginate and search admin requests"
```

### Task 3: Add frontend URL state and paginated API types

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/admin.ts`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/tests/admin.spec.ts`

**Interfaces:**
- Consumes: backend JSON contract from Tasks 1–2.
- Produces: `AdminRequestPage`, `AdminRequestQuery`, `readAdminQuery`, `writeAdminQuery`, and `adminApi.list(query, signal)`.

- [ ] **Step 1: Change the Playwright fixture to require server parameters**

Update the admin request route mock to inspect the URL and return:

```ts
{
  items: requests.slice(offset, offset + pageSize),
  page,
  pageSize,
  total: requests.length,
  summary: { active: 2, new: 1, overdue: 1, unassigned: 2, done: 1 },
}
```

Add assertions that filling search eventually sends `q=подъезд+2`, moving to page 2 sends `page=2`, and changing status removes `page` from the URL.

- [ ] **Step 2: Verify the frontend test fails**

Run: `cd frontend && pnpm test:e2e tests/admin.spec.ts --grep 'pagination|filters in the URL'`

Expected: FAIL because the API expects an array and the page has no pagination.

- [ ] **Step 3: Define frontend contracts**

Add:

```ts
export interface AdminRequestSummary { active: number; new: number; overdue: number; unassigned: number; done: number }
export interface AdminRequestPage { items: RequestItem[]; page: number; pageSize: 20 | 50 | 100; total: number; summary: AdminRequestSummary }
```

Extend `AdminFilters` with `page: number` and `pageSize: 20 | 50 | 100`. Keep UI values lower camel case and map them explicitly to backend uppercase enum values in `api.ts`.

- [ ] **Step 4: Implement typed URL parsing and writing**

Add pure helpers in `admin.ts` that validate known queues/sorts/page sizes and default invalid URL values. `writeAdminQuery(current, patch)` must delete default values and set `page=1` whenever a non-page list parameter changes.

- [ ] **Step 5: Change `adminApi.list`**

Accept the complete query object, append page/search/filter/sort values, and include local-day `visitStart`/`visitEnd` only for `visitToday`:

```ts
list: (query: AdminFilters, signal?: AbortSignal) =>
  call<AdminRequestPage>(`/admin/requests?${adminRequestParams(query)}`, { signal })
```

- [ ] **Step 6: Build and run the focused test**

Run: `cd frontend && pnpm build`

Then run the Playwright command from Step 2. The E2E test may remain red only for missing visual controls scheduled in Task 4; the type/API portion must compile and its intercepted request assertions must pass.

- [ ] **Step 7: Commit contract and URL state**

```bash
git add frontend/src/types.ts frontend/src/admin.ts frontend/src/api.ts frontend/tests/admin.spec.ts
git commit -m "feat: add admin pagination contract"
```

### Task 4: Build the operational request workspace

**Files:**
- Create: `frontend/src/components/AdminRequestSummary.tsx`
- Create: `frontend/src/components/AdminRequestFilters.tsx`
- Create: `frontend/src/components/AdminRequestList.tsx`
- Create: `frontend/src/components/AdminPagination.tsx`
- Modify: `frontend/src/pages/Admin.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/tests/admin.spec.ts`
- Test: `frontend/tests/responsive-layout.spec.ts`

**Interfaces:**
- Consumes: paginated contracts and URL helpers from Task 3.
- Produces: focused presentational components driven only by props/callbacks; `Admin.tsx` remains the sole list-data owner.

- [ ] **Step 1: Add failing workspace behavior tests**

Assert that desktop renders one semantic table and no `.admin-request-card`, mobile renders cards and hides the table, page controls show `Показано 21–25 из 25`, and page-size selection resets to page 1. Add a delayed/failing second API response and assert the old row remains visible beside a `Повторить` action.

In `responsive-layout.spec.ts`, visit `/max/admin` at 320 px with a paginated fixture and assert `document.documentElement.scrollWidth <= window.innerWidth`.

- [ ] **Step 2: Verify workspace tests fail**

Run: `cd frontend && pnpm test:e2e tests/admin.spec.ts tests/responsive-layout.spec.ts --grep 'workspace|pagination|mobile admin'`

Expected: FAIL because the current page always renders cards, blanks/replaces state during loads, and has no pagination.

- [ ] **Step 3: Extract summary and filters**

`AdminRequestSummary` renders Active, New, Overdue, Unassigned, Done buttons from server values. `AdminRequestFilters` owns only its debounced input draft; after 300 ms it calls `onChange({ query })`. Render organization, kind, status, and sort inside a disclosure and emit removable active-filter chips.

- [ ] **Step 4: Build desktop table and mobile cards**

`AdminRequestList` renders the same `items` twice with CSS-controlled visibility: a semantic table for desktop and accessible links/cards for mobile. Include number, emergency/overdue text marker, address and description, status badge, assignee fallback `Без исполнителя`, deadline label, and link to `/admin/requests/:id`.

- [ ] **Step 5: Add pagination controls**

`AdminPagination` computes `pageCount = Math.ceil(total/pageSize)`, a compact numbered window around the current page, range start/end, disabled Previous/Next buttons, and the exact page-size options 20/50/100. On an empty beyond-last response with nonzero total, `Admin.tsx` replaces the URL with the last valid page once.

- [ ] **Step 6: Simplify `Admin.tsx` orchestration**

Remove client-side `filterAdminRequests` and all configuration forms. Keep the last successful `AdminRequestPage` while a new request is in flight. Abort stale requests, expose a busy overlay/`aria-busy`, and preserve the page on refresh errors. Add an administrator-only link to `/admin/settings/users`.

- [ ] **Step 7: Add responsive and state styles**

At desktop widths show `.admin-request-table` and hide `.admin-request-cards`; below 700 px reverse them. Style active filter chips, fixed-width numeric/status cells, non-color emergency/overdue markers, and pagination wrapping. Remove obsolete `.admin-management`, `.admin-user-*`, and old list-card desktop declarations after their consumers move.

- [ ] **Step 8: Run frontend verification**

Run: `cd frontend && pnpm build`

Then: `cd frontend && pnpm test:e2e tests/admin.spec.ts tests/responsive-layout.spec.ts`

Expected: PASS.

- [ ] **Step 9: Commit the workspace UI**

```bash
git add frontend/src/components/AdminRequestSummary.tsx frontend/src/components/AdminRequestFilters.tsx frontend/src/components/AdminRequestList.tsx frontend/src/components/AdminPagination.tsx frontend/src/pages/Admin.tsx frontend/src/styles.css frontend/tests/admin.spec.ts frontend/tests/responsive-layout.spec.ts
git commit -m "feat: build paginated request workspace"
```

### Task 5: Separate administrator settings

**Files:**
- Create: `frontend/src/pages/AdminSettings.tsx`
- Create: `frontend/src/components/settings/UserSettings.tsx`
- Create: `frontend/src/components/settings/OrganizationSettings.tsx`
- Create: `frontend/src/components/settings/RoutingSettings.tsx`
- Modify: `frontend/src/components/RoutingAdmin.tsx`
- Modify: `frontend/src/main.tsx`
- Modify: `frontend/src/styles.css`
- Test: `frontend/tests/admin.spec.ts`

**Interfaces:**
- Consumes: existing `adminApi`, `routingAdminApi`, authentication context, and wildcard category behavior from the routing plan.
- Produces: administrator-only routes `/admin/settings/users`, `/admin/settings/organizations`, and `/admin/settings/routing`.

- [ ] **Step 1: Add failing access/navigation tests**

For an admin, assert the workspace has a Settings link and each URL exposes exactly one matching heading. For a manager, assert no Settings link exists and direct navigation to `/max/admin/settings/users` redirects to `/max/admin`. Assert the routing category control contains `Любая категория` with value `*` and helper text stating exact rules take priority.

- [ ] **Step 2: Verify settings tests fail**

Run: `cd frontend && pnpm test:e2e tests/admin.spec.ts --grep 'settings'`

Expected: FAIL because no settings routes exist and configuration remains embedded in `/admin`.

- [ ] **Step 3: Add an administrator route guard and tab shell**

Add `RequireAdmin` beside `RequireOrganizationRole` in `main.tsx`. Register the three concrete settings routes and redirect `/admin/settings` to `/admin/settings/users`. `AdminSettings` renders a back link, heading, and `NavLink` tabs before the selected child component.

- [ ] **Step 4: Move user access management**

Move role checkboxes and organization membership selectors from old `Admin.tsx` into `UserSettings`. Load users and organizations locally; disable only the row being saved; keep a row-level error and retry. Preserve the warning when a manager has no organization.

- [ ] **Step 5: Move organization management**

`OrganizationSettings` loads organizations and organization types. Its create form sends `{name, type, active: true}` through `routingAdminApi.saveOrganization`, so backend validation receives a real type. Render existing organizations in a readable list with active state and contact fields.

- [ ] **Step 6: Move responsibility rules**

`RoutingSettings` loads houses, organizations, types, and rules. Use a select for category with `*` first and the keys from `categories` after it; include place, urgency, role, source, and active controls. Label `*` as `Любая категория` and show `Точное правило категории имеет приоритет`.

Keep `RoutingAdmin` only for manual routing inside a `ROUTING_REQUIRED` request detail. Remove its no-`item` settings branch and associated combined grid.

- [ ] **Step 7: Style settings independently**

Add tab, section, list, and form styles under `.admin-settings-*`. Ensure forms become a single column below 700 px and all controls retain at least a 44 px touch target.

- [ ] **Step 8: Run settings and complete frontend tests**

Run: `cd frontend && pnpm build`

Then: `cd frontend && pnpm test:e2e tests/admin.spec.ts`

Then: `cd frontend && pnpm test:e2e`

Expected: PASS.

- [ ] **Step 9: Commit settings separation**

```bash
git add frontend/src/pages/AdminSettings.tsx frontend/src/components/settings/UserSettings.tsx frontend/src/components/settings/OrganizationSettings.tsx frontend/src/components/settings/RoutingSettings.tsx frontend/src/components/RoutingAdmin.tsx frontend/src/main.tsx frontend/src/styles.css frontend/tests/admin.spec.ts
git commit -m "feat: separate admin settings workspace"
```

### Task 6: End-to-end verification and documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/implementation-plan.md`

**Interfaces:**
- Consumes: completed routing and workspace plans.
- Produces: operator-facing API/UI documentation and final verification evidence.

- [ ] **Step 1: Update documentation**

Document the paginated response, accepted query parameters, `/admin/settings/**` routes, administrator-only configuration, and manager-visible queue behavior. Remove statements describing an all-in-one admin page or client-only filtering.

- [ ] **Step 2: Run formatting and static builds**

Run: `cd backend && gofmt -w domain/models.go controller/admin.go controller/admin_requests.go controller/admin_requests_test.go repository/request_access.go repository/admin_requests_integration_test.go service/routing.go service/routing_test.go service/request_test.go`

Run: `cd backend && go test ./... -count=1`

Run: `cd frontend && pnpm build`

Expected: all commands exit 0.

- [ ] **Step 3: Run focused user-flow tests**

Run: `cd frontend && pnpm test:e2e tests/admin.spec.ts tests/request-workflow.spec.ts tests/responsive-layout.spec.ts`

Expected: PASS with no horizontal-overflow assertion failures.

- [ ] **Step 4: Inspect the final diff**

Run: `git diff --check && git status --short`

Confirm there are no debug logs, generated screenshots, test-result directories, or unrelated file changes.

- [ ] **Step 5: Commit documentation**

```bash
git add README.md docs/implementation-plan.md
git commit -m "docs: describe admin request workspace"
```
