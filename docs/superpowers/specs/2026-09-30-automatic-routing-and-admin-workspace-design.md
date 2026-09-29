# Automatic Request Routing and Admin Workspace Design

## Goal

Remove routine manual moderation from request creation and turn the manager/admin area into a scalable operational workspace.

The finished flow must:

- route a request immediately when an unambiguous exact or fallback rule exists;
- send common-property requests to the configured management organization for the house;
- preserve specialist destinations for resource supply, capital repair, municipal territory, and other specifically configured categories;
- retain manual routing only when no applicable primary recipient exists or equally specific primary rules conflict;
- provide server-side request search, filtering, sorting, summaries, and pagination;
- separate the operational request queue from administrator-only configuration.

## Scope

This change covers request routing, the manager/admin request list API, the request workspace UI, administrator settings navigation, and focused automated tests.

It does not introduce external dispatch integrations, automatic discovery of a management organization from GIS Housing data, saved personal filter presets, bulk request actions, or changes to the request detail workflow.

## Routing Model

### Rule priority

Responsibility rules continue to be scoped to a house, validity period, organization role, place, and urgency. The `category` field additionally accepts `*`, meaning any category.

For each organization role, applicable rules are ranked by specificity:

1. exact category, exact place, exact urgency;
2. exact category with fewer optional constraints;
3. wildcard category with exact place and/or urgency;
4. wildcard category without optional constraints.

Category specificity outranks place and urgency specificity. Place outranks urgency. This ensures that a configured specialist rule such as `HEATING -> resource supplier` is selected before the generic `* + COMMON_PROPERTY -> management organization` fallback.

The resolver selects a rule only when exactly one rule has the highest specificity for a role. An absent or tied primary rule returns `ErrRoutingRequired`. Contractor and escalation ambiguity does not prevent primary delivery; those optional roles are omitted when their highest-ranked result is ambiguous.

### Default house routes

Migration 013 adds wildcard primary rules for the configured demo houses:

- `* + COMMON_PROPERTY -> configured management organization`;
- `* + APARTMENT -> configured management organization`;
- `* + YARD -> configured management organization`.

Existing exact-category rules remain unchanged and take priority. `RESOURCE_INPUT` and `CITY_TERRITORY` do not silently fall back to the management organization; they require a configured specialist rule. This avoids sending a request to an organization that is not responsible for it.

The admin routing settings allow `*` as the explicit “Any category” choice and explain that exact rules override it. Existing duplicate/overlap validation remains in force for rules with the same house, role, category, place, urgency, and intersecting validity period.

### Request creation

Request creation keeps its current validation and classification. The resolved route is stored as the immutable request snapshot. A successfully resolved request starts as `CREATED` and immediately becomes visible to the destination organization. `ROUTING_REQUIRED` is used only for a genuinely missing or ambiguous primary route.

The routing reason identifies whether the chosen rule was exact or a house fallback, including category and place. Manager notifications continue to be sent only after the request transaction commits.

## Request List API

### Endpoint

`GET /api/admin/requests` accepts:

- `page`: one-based page number, default `1`;
- `pageSize`: `20`, `50`, or `100`, default `20`;
- `q`: trimmed search text;
- `queue`: `ACTIVE`, `NEW`, `IN_WORK`, `UNASSIGNED`, `MINE`, `VISIT_TODAY`, `OVERDUE`, `DONE`, `EMERGENCY`, or `ALL`;
- `status`: an optional request status;
- `kind`: an optional request kind;
- `organization`: an optional organization ID, honored for administrators and constrained to the manager's own accessible scope;
- `sort`: `PRIORITY`, `NEWEST`, or `DEADLINE`.

Queue semantics preserve the existing workflow: `ACTIVE` excludes `CLOSED` and
`REJECTED`; `NEW` contains `ROUTING_REQUIRED`, `CREATED`, and `SENT`;
`IN_WORK` contains `ACCEPTED` and `IN_PROGRESS`; `DONE` contains `CLOSED` and
`REJECTED`; and the other queues apply their named condition while excluding
terminal requests. `MINE` means assigned to the current user, not merely
assigned to somebody. `VISIT_TODAY` uses the user's local-day boundaries sent
as validated RFC 3339 `visitStart` and `visitEnd` values.

Unknown enum values, invalid page values, and unsupported page sizes return a validation error rather than silently changing behavior.

The response shape is:

```json
{
  "items": [],
  "page": 1,
  "pageSize": 20,
  "total": 186,
  "summary": {
    "active": 42,
    "new": 8,
    "overdue": 5,
    "unassigned": 3,
    "done": 144
  }
}
```

Search matches request number, address snapshot, description, primary organization snapshot, contractor organization snapshot, and assigned employee name case-insensitively. Every whitespace-separated search term must occur somewhere in the combined searchable fields, matching the current interface behavior. All filtering and access restrictions are applied in SQL before counting and pagination. Search parameters remain bound query values; sort options map to fixed SQL fragments.

`total` reflects every active filter including the selected queue and search. `summary` reflects the accessible organization scope and optional organization selector, but ignores queue, status, kind, and search so the dashboard counters remain stable while the user explores the list.

The repository returns the items, filtered total, and summary from one read operation. The access predicate remains identical for rows and counts: an administrator may see every request; a manager may see only requests where their organization is primary, contractor, or the assignee's organization.

## Operational Workspace

### Navigation and responsibilities

`/admin` becomes the operational workspace for both managers and administrators. It contains no user, organization, or rule-editing forms.

Administrators see a “Settings” link leading to `/admin/settings`. Managers do not see the link. The frontend redirects a manager who enters the settings URL back to `/admin`, while every administrator-only API continues to return `403` for a direct manager request.

### Desktop layout

The page contains, in order:

1. workspace title, last-updated time, refresh action, and administrator settings link;
2. summary buttons for Active, New, Overdue, Unassigned, and Done;
3. queue tabs for common operational views;
4. a persistent search field and a compact filter/sort toolbar;
5. a dense request table with number, priority, address/topic, status, assignee, deadline, and row navigation;
6. result range, page-size selector, and numbered pagination.

Less common filters are grouped behind a “Filters” disclosure. Active filters are visible as removable chips, and “Reset” clears them all. Emergency and overdue rows are visually distinct without relying on color alone.

### Mobile layout

At mobile widths, the same response is rendered as stacked request cards. Essential fields remain visible; secondary information is collapsed. Pagination uses Previous/Next plus the current page indicator, and no horizontal scrolling is introduced.

### URL and request behavior

Queue, search, filters, sorting, page, and page size are represented in the URL. Changing queue, search, filter, sort, or page size resets `page` to `1`. Browser Back/Forward restores the prior list state.

Search is debounced by approximately 300 milliseconds. An in-flight request is cancelled when list parameters change. Refresh reuses the current parameters.

While a new page loads, the existing results remain visible with a busy indication. A failed refresh or page change keeps the existing list and shows a retry action. A first-load failure uses the full error state. An empty result describes the applied filters and offers reset.

The API returns an empty `items` array when a requested page is beyond the last page while preserving the correct `total`. If `total` is nonzero, the frontend replaces the URL with the last valid page and refetches once. This handles deletion or filter changes without hiding the real total or creating redirect behavior in the API.

## Administrator Settings

`/admin/settings` uses three sections, represented as URL-addressable tabs:

- `/admin/settings/users`: roles and organization membership;
- `/admin/settings/organizations`: organization creation and organization records;
- `/admin/settings/routing`: responsibility rules, including the Any category fallback.

Each section owns its loading, error, and mutation state so a failure in one section does not disable unrelated settings. Existing APIs remain unless the UI needs a small focused read endpoint; request-list pagination does not alter settings APIs.

The former `RoutingAdmin` content is split into focused components rather than one combined form. Manual routing on a request detail page remains available only for `ROUTING_REQUIRED` requests and is not moved into settings.

## Component Boundaries

Backend:

- `service/routing.go`: rule applicability and specificity ranking;
- `repository/routing.go`: fetch exact and wildcard candidate rules;
- migration 013: wildcard demo-house fallback rules;
- request-list query/DTO types in the domain or service boundary;
- controller parsing and validation for list parameters;
- repository query that applies access, filters, counts, sorting, and pagination.

Frontend:

- `Admin.tsx`: page orchestration and URL state only;
- request summary, queue navigation, search/filter controls, desktop table, mobile cards, and pagination as focused components;
- `AdminSettings.tsx`: settings route and tab navigation;
- separate Users, Organizations, and Routing settings components;
- typed paginated response and query construction in `api.ts`.

Components receive typed values and callbacks; they do not issue duplicate list requests or maintain competing copies of URL filter state.

## Testing

Routing service tests prove:

- an exact specialist rule beats the wildcard management-company fallback;
- a common-property request with no exact category rule uses the house management organization;
- equal highest-specificity primary rules still require manual routing;
- optional contractor ambiguity does not block a unique primary route.

Repository/integration tests prove wildcard rule retrieval, seeded fallback presence, access isolation, every queue predicate, combined search/filter behavior, stable sorting, page boundaries, totals, and summary counts.

Controller tests prove defaults, invalid parameters, page-size allow-listing, manager organization constraints, and administrator access.

Frontend tests prove debounced search, URL restoration, page reset after filter changes, page-size changes, retry without blanking existing data, desktop table behavior, mobile card behavior without overflow, settings access, and the three settings tabs.

## Rollout and Compatibility

Migration 013 is additive and idempotent. Existing requests and route snapshots are untouched. Existing exact responsibility rules remain authoritative.

The admin request-list response changes from an array to a paginated object. Backend and frontend are deployed together, so no compatibility adapter is required inside this repository. The resident request APIs are unchanged.

## Success Criteria

- A new common-property request for a configured house reaches its management organization without administrator action.
- Specialist rules continue to win over fallback rules.
- Manual routing appears only for missing or ambiguous primary responsibility.
- Managers and administrators can search and navigate large request sets without loading every row.
- List state is reproducible from the URL.
- Administrative configuration is absent from the operational queue and inaccessible to managers.
- The workspace remains usable at 320 px and at desktop widths without horizontal page overflow.
