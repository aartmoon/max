package db

import (
	"bytes"
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"tvoydom/gar-init/internal/model"
)

//go:embed fixtures/*.csv
var demoFixtures embed.FS

func loadDemoFixtures(ctx context.Context, tx pgx.Tx) error {
	for _, family := range model.Families() {
		path := "fixtures/" + family.Key + ".csv"
		data, err := demoFixtures.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read demo fixture %s: %w", path, err)
		}
		want, err := csvDataRows(data)
		if err != nil {
			return fmt.Errorf("validate demo fixture %s: %w", path, err)
		}
		columns := make([]string, len(family.Columns()))
		for index, column := range family.Columns() {
			columns[index] = pgx.Identifier{column}.Sanitize()
		}
		copySQL := fmt.Sprintf(
			"COPY %s (%s) FROM STDIN WITH (FORMAT csv, HEADER true)",
			pgx.Identifier{family.Table}.Sanitize(), strings.Join(columns, ","),
		)
		got, err := tx.Conn().PgConn().CopyFrom(ctx, bytes.NewReader(data), copySQL)
		if err != nil {
			return fmt.Errorf("load demo fixture %s: %w", family.Table, err)
		}
		if got.RowsAffected() != int64(want) {
			return fmt.Errorf("load demo fixture %s: copied %d rows, expected %d", family.Table, got.RowsAffected(), want)
		}
	}
	return nil
}

func csvDataRows(data []byte) (int, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}
	if len(header) == 0 {
		return 0, fmt.Errorf("empty header")
	}
	count := 0
	for {
		_, err := reader.Read()
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return 0, err
		}
		count++
	}
}
