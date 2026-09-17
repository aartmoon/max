package importer

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tvoydom/gar-init/internal/model"
)

type BatchWriter func(context.Context, model.Family, [][]any) error
type ParseFile func(context.Context, BatchWriter) (int64, error)

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func Parse(
	ctx context.Context,
	r io.Reader,
	family model.Family,
	batchSize int,
	logEvery int,
	write BatchWriter,
	progress func(int64),
) (int64, error) {
	if batchSize <= 0 {
		return 0, fmt.Errorf("batch size must be positive")
	}
	decoder := xml.NewDecoder(r)
	batch := make([][]any, 0, batchSize)
	var count int64
	for {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("decode %s XML at offset %d: %w", family.Key, decoder.InputOffset(), err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if _, ok := family.Elements[strings.ToUpper(start.Name.Local)]; !ok {
			continue
		}
		row, err := parseRow(start, family)
		if err != nil {
			return count, fmt.Errorf("parse %s XML at offset %d: %w", family.Key, decoder.InputOffset(), err)
		}
		batch = append(batch, row)
		count++
		if logEvery > 0 && count%int64(logEvery) == 0 {
			progress(count)
		}
		if len(batch) == batchSize {
			if err := write(ctx, family, batch); err != nil {
				return count, fmt.Errorf("copy %s batch ending at row %d: %w", family.Key, count, err)
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := write(ctx, family, batch); err != nil {
			return count, fmt.Errorf("copy final %s batch ending at row %d: %w", family.Key, count, err)
		}
	}
	return count, nil
}

func parseRow(start xml.StartElement, family model.Family) ([]any, error) {
	attributes := make(map[string]string, len(start.Attr))
	for _, attribute := range start.Attr {
		attributes[strings.ToUpper(attribute.Name.Local)] = strings.TrimSpace(attribute.Value)
	}
	row := make([]any, 0, len(family.Fields)+1)
	for _, field := range family.Fields {
		value, err := convert(attributes[field.Attr], field)
		if err != nil {
			return nil, fmt.Errorf("attribute %s: %w", field.Attr, err)
		}
		row = append(row, value)
	}
	raw, err := json.Marshal(attributes)
	if err != nil {
		return nil, fmt.Errorf("encode raw attributes: %w", err)
	}
	return append(row, raw), nil
}

func convert(raw string, field model.Field) (any, error) {
	if raw == "" {
		return nil, nil
	}
	switch field.Kind {
	case model.Text:
		return raw, nil
	case model.Int64:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid bigint %q", raw)
		}
		return value, nil
	case model.Int:
		value, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", raw)
		}
		return value, nil
	case model.UUID:
		if !uuidPattern.MatchString(raw) {
			return nil, fmt.Errorf("invalid UUID %q", raw)
		}
		return strings.ToLower(raw), nil
	case model.Date:
		value, err := time.Parse(time.DateOnly, raw)
		if err != nil {
			return nil, fmt.Errorf("invalid date %q", raw)
		}
		return value, nil
	case model.Bool:
		switch strings.ToLower(raw) {
		case "1", "true":
			return true, nil
		case "0", "false":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean %q", raw)
		}
	default:
		return nil, fmt.Errorf("unsupported value kind %d", field.Kind)
	}
}
