package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tvoydom/domain"
)

func TestHouseProfileDatabaseRoundTrip(t *testing.T) {
	repo := profileTestRepository(t)
	ctx := context.Background()
	if _, err := repo.Pool.Exec(ctx, `INSERT INTO houses(id) VALUES(42)`); err != nil {
		t.Fatal(err)
	}
	date := time.Date(2015, 4, 22, 0, 0, 0, 0, time.UTC)
	fetched := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	profile := domain.HouseProfile{
		HouseID: "42", GISHouseGUID: "105a84e6-01b5-4bbb-a26d-e5ec59bee321", GISHouseType: "1",
		CadastralNumber: p("77:01:0001046:1013"), RawPayload: json.RawMessage(`{"detail":true}`),
		SquarePayload: json.RawMessage(`{"square":true}`), SquareSummaryAvailable: true, FetchedAt: fetched,
		Characteristics: domain.HouseCharacteristics{
			HouseTypeCode: p("1"), HouseType: p("Многоквартирный"), Status: p("APPROVED"),
			ProjectSeries: p("Индивидуальный проект"), Condition: p("Исправный"), LifecycleStage: p("Эксплуатация"),
			YearBuilt: p(1870), OperationYear: p(1870), ReconstructionYear: p(1870),
			DeteriorationPercent: p(66.0), DeteriorationDate: &date, WallMaterial: p("Стены кирпичные"),
			EnergyEfficiency: p("B"), TotalArea: p(6720.8), LivingArea: p(2405.3), NonResidentialArea: p(4023.6),
			ResidentialPremises: p(15), ResidentialPremisesArea: p(2405.3), ResidentialPremisesWithRealty: p(15),
			ResidentialPremisesWithRealtyArea: p(2405.3), NonResidentialPremises: p(31), NonResidentialPremisesArea: p(4176.4),
			NonResidentialPremisesNotCommon: p(29), NonResidentialPremisesNotCommonArea: p(4023.6),
			Floors: p(7), Entrances: p(2), OwnersOrShares: p(61),
		},
		Management: domain.HouseManagement{
			Method: p("УО"), OrganizationGUID: p("org-guid"), ShortName: p("ГБУ Жилищник"), FullName: p("Полное имя"),
			Address: p("Москва"), Phone: p("74950000000"), Website: p("https://example.test"), OrganizationType: p("L"),
			RegistryOrganizationGUID: p("registry-guid"), INN: p("7700000000"), OGRN: p("5147746267906"),
			Chief: p("Иванов Иван Иванович"), ContractStart: &date,
		},
	}
	if err := repo.SaveHouseProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetHouseProfile(ctx, "42")
	if err != nil {
		t.Fatal(err)
	}
	if got.Characteristics.HouseType == nil || *got.Characteristics.HouseType != "Многоквартирный" || got.Characteristics.ResidentialPremises == nil || *got.Characteristics.ResidentialPremises != 15 {
		t.Fatalf("characteristics did not round trip: %+v", got.Characteristics)
	}
	if got.Management.OGRN == nil || *got.Management.OGRN != "5147746267906" || got.Management.ContractStart == nil || !got.Management.ContractStart.Equal(date) {
		t.Fatalf("management did not round trip: %+v", got.Management)
	}
	if !got.SquareSummaryAvailable || string(got.RawPayload) != `{"detail": true}` || string(got.SquarePayload) != `{"square": true}` || !got.FetchedAt.Equal(fetched) {
		t.Fatalf("payload metadata did not round trip: %+v", got)
	}
	if len(got.DataSources) != 2 || !got.DataSources[1].Available {
		t.Fatalf("sources not restored: %+v", got.DataSources)
	}
}

func TestHouseProfileDatabaseRoundTripPreservesNulls(t *testing.T) {
	repo := profileTestRepository(t)
	ctx := context.Background()
	if _, err := repo.Pool.Exec(ctx, `INSERT INTO houses(id) VALUES(43)`); err != nil {
		t.Fatal(err)
	}
	profile := domain.HouseProfile{
		HouseID: "43", GISHouseGUID: "205a84e6-01b5-4bbb-a26d-e5ec59bee321", GISHouseType: "1",
		RawPayload: json.RawMessage(`{}`), FetchedAt: time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC),
	}
	if err := repo.SaveHouseProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetHouseProfile(ctx, "43")
	if err != nil {
		t.Fatal(err)
	}
	if got.Characteristics.Floors != nil || got.Management.Phone != nil || len(got.SquarePayload) != 0 || got.SquareSummaryAvailable {
		t.Fatalf("null values changed: %+v", got)
	}
	if len(got.DataSources) != 2 || got.DataSources[1].Available {
		t.Fatalf("unexpected optional source: %+v", got.DataSources)
	}
}

func profileTestRepository(t *testing.T) Postgres {
	t.Helper()
	databaseURL := os.Getenv("HOUSE_PROFILE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("HOUSE_PROFILE_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("house_profile_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	admin.Close()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ", public"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		cleanup, cleanupErr := pgxpool.New(context.Background(), databaseURL)
		if cleanupErr == nil {
			_, _ = cleanup.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
			cleanup.Close()
		}
	})
	if _, err := pool.Exec(ctx, `CREATE TABLE houses(id bigint PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, gisHouseProfilesMigration); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, enrichedGISHouseProfilesMigration); err != nil {
		t.Fatal(err)
	}
	return Postgres{Pool: pool}
}

func p[T any](value T) *T { return &value }
