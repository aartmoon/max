package gar

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ErrNoSupportedFiles = errors.New("GAR archive contains no supported XML files")

type Options struct {
	BatchSize int
}

type Store interface {
	UpsertAddressObjects(context.Context, []AddressObject) error
	UpsertHierarchy(context.Context, []HierarchyItem) error
	UpsertHouseTypes(context.Context, []HouseType) error
	UpsertParamTypes(context.Context, []ParamType) error
	UpsertParams(context.Context, []Param) error
	UpsertHouses(context.Context, []HouseRecord) error
	RefreshAddresses(context.Context) error
}

type AddressObject struct {
	ObjectID   int64
	ObjectGUID string
	Name       string
	TypeName   string
	Level      int
	IsActive   bool
	IsActual   bool
	UpdatedAt  time.Time
}

type HierarchyItem struct {
	ObjectID       int64
	ParentObjectID *int64
	IsActive       bool
	UpdatedAt      time.Time
}

type HouseType struct {
	ID        int
	Name      string
	ShortName string
	IsActive  bool
}

type ParamType struct {
	ID       int
	Name     string
	Code     string
	IsActive bool
}

type Param struct {
	ObjectID  int64
	TypeID    int
	Value     string
	UpdatedAt time.Time
}

type HouseRecord struct {
	ObjectID   int64
	ObjectGUID string
	HouseNum   string
	AddNum1    string
	AddNum2    string
	HouseType  *int
	AddType1   *int
	AddType2   *int
	IsActive   bool
	IsActual   bool
	UpdatedAt  time.Time
}

func ImportZip(ctx context.Context, r io.ReaderAt, size int64, store Store, opts Options) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return err
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 1000
	}
	recognized := 0
	for _, f := range zr.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if f.FileInfo().IsDir() || !strings.EqualFold(filepath.Ext(f.Name), ".xml") {
			continue
		}
		name := strings.ToUpper(filepath.Base(f.Name))
		rc, err := f.Open()
		if err != nil {
			return err
		}
		switch {
		case strings.HasPrefix(name, "AS_ADDR_OBJ_"):
			recognized++
			err = streamAddressObjects(ctx, rc, store, opts.BatchSize)
		case strings.HasPrefix(name, "AS_ADM_HIERARCHY_"):
			recognized++
			err = streamHierarchy(ctx, rc, store, opts.BatchSize)
		case strings.HasPrefix(name, "AS_HOUSE_TYPES_"):
			recognized++
			err = streamHouseTypes(ctx, rc, store, opts.BatchSize)
		case strings.HasPrefix(name, "AS_PARAM_TYPES_"):
			recognized++
			err = streamParamTypes(ctx, rc, store, opts.BatchSize)
		case strings.HasPrefix(name, "AS_HOUSES_PARAMS_") || strings.HasPrefix(name, "AS_PARAM_"):
			recognized++
			err = streamParams(ctx, rc, store, opts.BatchSize)
		case strings.HasPrefix(name, "AS_HOUSES_"):
			recognized++
			err = streamHouses(ctx, rc, store, opts.BatchSize)
		}
		closeErr := rc.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if recognized == 0 {
		return ErrNoSupportedFiles
	}
	return store.RefreshAddresses(ctx)
}

func streamAddressObjects(ctx context.Context, r io.Reader, store Store, batchSize int) error {
	dec := xml.NewDecoder(r)
	var batch []AddressObject
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "OBJECT" {
			continue
		}
		item := AddressObject{
			ObjectID:   attrInt64(start, "OBJECTID"),
			ObjectGUID: strings.ToLower(attr(start, "OBJECTGUID")),
			Name:       attr(start, "NAME"),
			TypeName:   attr(start, "TYPENAME"),
			Level:      attrInt(start, "LEVEL"),
			IsActive:   attr(start, "ISACTIVE") == "1",
			IsActual:   attr(start, "ISACTUAL") == "1",
			UpdatedAt:  attrDate(start, "UPDATEDATE"),
		}
		batch = append(batch, item)
		if len(batch) >= batchSize {
			if err := store.UpsertAddressObjects(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return store.UpsertAddressObjects(ctx, batch)
	}
	return nil
}

func streamHierarchy(ctx context.Context, r io.Reader, store Store, batchSize int) error {
	dec := xml.NewDecoder(r)
	var batch []HierarchyItem
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "ITEM" {
			continue
		}
		item := HierarchyItem{ObjectID: attrInt64(start, "OBJECTID"), IsActive: attr(start, "ISACTIVE") == "1", UpdatedAt: attrDate(start, "UPDATEDATE")}
		if v := attr(start, "PARENTOBJID"); v != "" {
			parent := parseInt64(v)
			item.ParentObjectID = &parent
		}
		batch = append(batch, item)
		if len(batch) >= batchSize {
			if err := store.UpsertHierarchy(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return store.UpsertHierarchy(ctx, batch)
	}
	return nil
}

func streamHouses(ctx context.Context, r io.Reader, store Store, batchSize int) error {
	dec := xml.NewDecoder(r)
	var batch []HouseRecord
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "HOUSE" {
			continue
		}
		item := HouseRecord{
			ObjectID:   attrInt64(start, "OBJECTID"),
			ObjectGUID: strings.ToLower(attr(start, "OBJECTGUID")),
			HouseNum:   attr(start, "HOUSENUM"),
			AddNum1:    attr(start, "ADDNUM1"),
			AddNum2:    attr(start, "ADDNUM2"),
			HouseType:  optionalInt(start, "HOUSETYPE"),
			AddType1:   optionalInt(start, "ADDTYPE1"),
			AddType2:   optionalInt(start, "ADDTYPE2"),
			IsActive:   attr(start, "ISACTIVE") == "1",
			IsActual:   attr(start, "ISACTUAL") == "1",
			UpdatedAt:  attrDate(start, "UPDATEDATE"),
		}
		batch = append(batch, item)
		if len(batch) >= batchSize {
			if err := store.UpsertHouses(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return store.UpsertHouses(ctx, batch)
	}
	return nil
}

func streamHouseTypes(ctx context.Context, r io.Reader, store Store, batchSize int) error {
	dec := xml.NewDecoder(r)
	var batch []HouseType
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "HOUSETYPE" {
			continue
		}
		batch = append(batch, HouseType{ID: attrInt(start, "ID"), Name: attr(start, "NAME"), ShortName: attr(start, "SHORTNAME"), IsActive: attr(start, "ISACTIVE") == "true"})
		if len(batch) >= batchSize {
			if err := store.UpsertHouseTypes(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return store.UpsertHouseTypes(ctx, batch)
	}
	return nil
}

func streamParamTypes(ctx context.Context, r io.Reader, store Store, batchSize int) error {
	dec := xml.NewDecoder(r)
	var batch []ParamType
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "PARAMTYPE" {
			continue
		}
		batch = append(batch, ParamType{ID: attrInt(start, "ID"), Name: attr(start, "NAME"), Code: attr(start, "CODE"), IsActive: attr(start, "ISACTIVE") == "true"})
		if len(batch) >= batchSize {
			if err := store.UpsertParamTypes(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return store.UpsertParamTypes(ctx, batch)
	}
	return nil
}

func streamParams(ctx context.Context, r io.Reader, store Store, batchSize int) error {
	dec := xml.NewDecoder(r)
	var batch []Param
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "PARAM" {
			continue
		}
		batch = append(batch, Param{ObjectID: attrInt64(start, "OBJECTID"), TypeID: attrInt(start, "TYPEID"), Value: attr(start, "VALUE"), UpdatedAt: attrDate(start, "UPDATEDATE")})
		if len(batch) >= batchSize {
			if err := store.UpsertParams(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return store.UpsertParams(ctx, batch)
	}
	return nil
}

func attr(start xml.StartElement, name string) string {
	for _, a := range start.Attr {
		if a.Name.Local == name {
			return strings.TrimSpace(a.Value)
		}
	}
	return ""
}

func attrInt(start xml.StartElement, name string) int {
	v, _ := strconv.Atoi(attr(start, name))
	return v
}

func attrInt64(start xml.StartElement, name string) int64 {
	return parseInt64(attr(start, name))
}

func parseInt64(value string) int64 {
	v, _ := strconv.ParseInt(value, 10, 64)
	return v
}

func optionalInt(start xml.StartElement, name string) *int {
	value := attr(start, name)
	if value == "" {
		return nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &v
}

func attrDate(start xml.StartElement, name string) time.Time {
	t, _ := time.Parse("2006-01-02", attr(start, name))
	return t
}
