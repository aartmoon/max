# Automatic Request Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Route common-property, apartment, and yard requests directly to the configured house management organization while preserving more specific specialist rules.

**Architecture:** Extend the existing responsibility-rule engine with the explicit wildcard category `*`. Rank exact category rules above wildcard fallbacks, seed per-house management fallbacks in migration 013, and retain `ROUTING_REQUIRED` only for missing or equally specific primary rules.

**Tech Stack:** Go 1.24, PostgreSQL/pgx, SQL migrations, Go `testing`.

**Spec:** `docs/superpowers/specs/2026-09-30-automatic-routing-and-admin-workspace-design.md`

## Global Constraints

- Exact category rules must always outrank wildcard category rules.
- Place specificity outranks urgency specificity within the same category specificity.
- Ambiguous optional contractor or escalation rules must not block a unique primary route.
- Existing requests and stored routing snapshots must not be rewritten.
- `RESOURCE_INPUT` and `CITY_TERRITORY` must not silently fall back to the house management organization.
- Production code is written only after its focused test fails for the expected reason.

---

### Task 1: Wildcard-aware routing specificity

**Files:**
- Modify: `backend/service/routing.go`
- Test: `backend/service/routing_test.go`

**Interfaces:**
- Consumes: `RouteQuery`, `domain.ResponsibilityRule`, and `ResponsibilityRepository.FindActiveRules`.
- Produces: `RoutingService.Resolve(context.Context, RouteQuery) (RouteDecision, error)` supporting `ResponsibilityRule.Category == "*"`.

- [ ] **Step 1: Add failing routing tests**

Add table-driven tests with these concrete rules:

```go
func TestRoutingUsesWildcardHouseFallbackWhenExactCategoryIsMissing(t *testing.T) {
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	rules := []domain.ResponsibilityRule{
		{ID: "fallback", Role: "PRIMARY", Category: "*", Place: "COMMON_PROPERTY", OrganizationID: "10", OrganizationName: "УК дома", Active: true},
	}
	got, err := (RoutingService{Repo: ruleRepoStub{rules: rules}}).Resolve(context.Background(), RouteQuery{Category: "OTHER", Place: "COMMON_PROPERTY", Urgency: "NORMAL", At: now})
	if err != nil || got.Primary == nil || got.Primary.ID != "fallback" {
		t.Fatalf("unexpected fallback decision: %+v, err=%v", got, err)
	}
	if !strings.Contains(got.Reason, "резервное правило") {
		t.Fatalf("fallback reason is not explicit: %q", got.Reason)
	}
}

func TestRoutingExactCategoryBeatsWildcardPlaceFallback(t *testing.T) {
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	rules := []domain.ResponsibilityRule{
		{ID: "fallback", Role: "PRIMARY", Category: "*", Place: "COMMON_PROPERTY", OrganizationID: "10", Active: true},
		{ID: "specialist", Role: "PRIMARY", Category: "HEATING", OrganizationID: "20", Active: true},
	}
	got, err := (RoutingService{Repo: ruleRepoStub{rules: rules}}).Resolve(context.Background(), RouteQuery{Category: "HEATING", Place: "COMMON_PROPERTY", Urgency: "NORMAL", At: now})
	if err != nil || got.Primary == nil || got.Primary.ID != "specialist" {
		t.Fatalf("exact rule did not win: %+v, err=%v", got, err)
	}
}
```

Extend the ambiguity test so two tied primary wildcard rules return `ErrRoutingRequired`, then add a case proving two tied contractor rules yield `Contractor == nil` while the unique primary remains selected.

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `cd backend && go test ./service -run 'TestRouting(UsesWildcard|ExactCategory|RequiresManual|Keeps)' -count=1`

Expected: the wildcard fallback test fails with `ErrRoutingRequired`, and the optional ambiguity test fails because the existing resolver selects no primary decision.

- [ ] **Step 3: Implement specificity ranking**

Replace the current applicability/scoring block with one that assigns non-overlapping weights:

```go
categoryScore := 0
switch rule.Category {
case query.Category:
	categoryScore = 4
case "*":
	categoryScore = 0
default:
	continue
}
score := categoryScore
if rule.Place != "" {
	if rule.Place != query.Place { continue }
	score += 2
}
if rule.Urgency != "" {
	if rule.Urgency != query.Urgency { continue }
	score++
}
```

Select primary only when its best slice has length one. For contractor and escalation, set the result only when their best slice has length one. Generate an exact-rule reason for an exact category and a reason containing `резервное правило дома` for `*`.

- [ ] **Step 4: Run service tests and verify GREEN**

Run: `cd backend && go test ./service -run 'Routing|Danger' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the resolver change**

```bash
git add backend/service/routing.go backend/service/routing_test.go
git commit -m "feat: add house fallback routing rules"
```

### Task 2: Retrieve and seed wildcard responsibility rules

**Files:**
- Create: `backend/repository/migrations/013_house_default_routes.sql`
- Modify: `backend/repository/postgres.go`
- Modify: `backend/repository/routing.go`
- Modify: `backend/repository/workflow_migration_integration_test.go`
- Test: `backend/repository/routing_integration_test.go`

**Interfaces:**
- Consumes: wildcard behavior from Task 1.
- Produces: `Postgres.FindActiveRules` returning exact and `*` candidates, and schema migration version 13.

- [ ] **Step 1: Add a failing repository integration test for candidate retrieval**

In `routing_integration_test.go`, create an isolated migrated schema using the same `WORKFLOW_TEST_DATABASE_URL` helper pattern as `workflow_migration_integration_test.go`. Insert one exact and one wildcard active rule for the same house, call:

```go
rules, err := repo.FindActiveRules(ctx, service.RouteQuery{
	HouseID: houseID, Category: "HEATING", Place: "COMMON_PROPERTY",
	Urgency: "NORMAL", At: time.Now().UTC(),
})
```

Assert both rule IDs are returned. Name the test `TestFindActiveRulesReturnsExactAndWildcardCandidates`.

- [ ] **Step 2: Verify the repository test fails for the expected reason**

Run: `cd backend && WORKFLOW_TEST_DATABASE_URL="$DATABASE_URL" go test ./repository -run TestFindActiveRulesReturnsExactAndWildcardCandidates -count=1`

Expected: FAIL because `FindActiveRules` filters on `r.category=$2`.

- [ ] **Step 3: Update the repository query**

Change the category predicate to:

```sql
WHERE r.house_id=$1 AND r.category IN ($2, '*') AND r.active
```

Keep the existing validity, place, urgency, and bound-parameter predicates.

- [ ] **Step 4: Verify candidate retrieval passes**

Run the command from Step 2.

Expected: PASS.

- [ ] **Step 5: Add the version-13 migration and a failing seed assertion**

Add this idempotent seed shape in `013_house_default_routes.sql`:

```sql
INSERT INTO house_responsibility_rules(
  house_id,organization_id,category,place,urgency,organization_role,
  valid_from,source,source_url,is_demo
)
SELECT h.id,1007,'*',v.place,'','PRIMARY',
       TIMESTAMPTZ '2026-09-30 00:00:00+00',
       'Резервное правило управляющей организации для дома',
       'https://www.mos.ru/',true
FROM houses h
CROSS JOIN (VALUES ('COMMON_PROPERTY'),('APARTMENT'),('YARD')) AS v(place)
WHERE h.address ILIKE '%Арбат%'
AND NOT EXISTS (
  SELECT 1 FROM house_responsibility_rules r
  WHERE r.house_id=h.id AND r.category='*' AND r.place=v.place
    AND r.organization_role='PRIMARY' AND r.active
);
```

Extend `TestWorkflowMigrationsPreserveLegacyRequestAndSeedOrganizations` to assert every migrated Arbat house has three active wildcard primary rows for organization `1007` and none for `RESOURCE_INPUT` or `CITY_TERRITORY`.

- [ ] **Step 6: Register migration 013**

Embed `migrations/013_house_default_routes.sql` as `houseDefaultRoutesMigration`, check `schema_migrations` version 13 after version 12, execute it once, and insert version 13 in the same transaction.

- [ ] **Step 7: Run migration and repository tests**

Run: `cd backend && go test ./repository -count=1`

Expected: PASS; database-dependent tests may report SKIP only when `WORKFLOW_TEST_DATABASE_URL` is absent.

- [ ] **Step 8: Commit persistence changes**

```bash
git add backend/repository/migrations/013_house_default_routes.sql backend/repository/postgres.go backend/repository/routing.go backend/repository/routing_integration_test.go backend/repository/workflow_migration_integration_test.go
git commit -m "feat: seed default management routes"
```

### Task 3: Prove request creation bypasses routine moderation

**Files:**
- Modify: `backend/service/request_test.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: wildcard resolver and repository behavior from Tasks 1–2.
- Produces: regression coverage for `RequestService.Create` and updated operator documentation.

- [ ] **Step 1: Add the failing request-creation regression test**

Create a resolver stub backed by `RoutingService` and one wildcard common-property rule, then call `Create` with an `OTHER` description and `Place: "COMMON_PROPERTY"`. Assert:

```go
if r.Status != "CREATED" || r.PrimaryOrganizationID != "10" || r.ResponsibleOrganization != "УК дома" {
	t.Fatalf("request was not routed directly: %+v", r)
}
if !strings.Contains(r.RoutingReason, "резервное правило") {
	t.Fatalf("unexpected routing reason: %q", r.RoutingReason)
}
```

Add a second case using `CITY_TERRITORY` with no matching rule and assert `Status == "ROUTING_REQUIRED"`.

- [ ] **Step 2: Run the regression test**

Run: `cd backend && go test ./service -run TestCreateRoutesCommonPropertyThroughHouseFallback -count=1`

Expected: PASS after Tasks 1–2; if it fails, fix production routing rather than weakening the assertions.

- [ ] **Step 3: Update routing documentation**

Replace README statements that say every missing exact category needs manual routing. Document the priority as “exact category → wildcard house/place rule → manual routing,” and state that city/resource locations still require a specialist rule.

- [ ] **Step 4: Run the complete backend suite**

Run: `cd backend && go test ./... -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the routing regression and documentation**

```bash
git add backend/service/request_test.go README.md
git commit -m "test: cover automatic request routing"
```

