package model

import (
	"path/filepath"
	"sort"
	"strings"
)

type ValueKind uint8

const (
	Text ValueKind = iota
	Int64
	Int
	UUID
	Date
	Bool
)

type Field struct {
	Column string
	Attr   string
	Kind   ValueKind
}

type Family struct {
	Key      string
	Prefix   string
	Table    string
	Elements map[string]struct{}
	Fields   []Field
	Order    int
}

func (f Family) Columns() []string {
	columns := make([]string, 0, len(f.Fields)+1)
	for _, field := range f.Fields {
		columns = append(columns, field.Column)
	}
	return append(columns, "raw_attributes")
}

var families = buildFamilies()

func Families() []Family {
	out := make([]Family, len(families))
	copy(out, families)
	return out
}

func MatchFile(name string) (Family, bool) {
	base := strings.ToUpper(filepath.Base(name))
	if !strings.HasSuffix(base, ".XML") {
		return Family{}, false
	}
	candidates := Families()
	sort.SliceStable(candidates, func(i, j int) bool {
		return len(candidates[i].Prefix) > len(candidates[j].Prefix)
	})
	for _, family := range candidates {
		if strings.HasPrefix(base, family.Prefix) {
			return family, true
		}
	}
	return Family{}, false
}

func buildFamilies() []Family {
	commonParams := []Field{
		f("id", "ID", Int64), f("object_id", "OBJECTID", Int64), f("change_id", "CHANGEID", Int64),
		f("change_id_end", "CHANGEIDEND", Int64), f("type_id", "TYPEID", Int), f("value", "VALUE", Text),
		f("update_date", "UPDATEDATE", Date), f("start_date", "STARTDATE", Date), f("end_date", "ENDDATE", Date),
	}
	objectFields := []Field{
		f("id", "ID", Int64), f("object_id", "OBJECTID", Int64), f("object_guid", "OBJECTGUID", UUID),
		f("change_id", "CHANGEID", Int64), f("name", "NAME", Text), f("type_name", "TYPENAME", Text),
		f("level", "LEVEL", Int), f("operation_type_id", "OPERTYPEID", Int), f("previous_id", "PREVID", Int64),
		f("next_id", "NEXTID", Int64), f("update_date", "UPDATEDATE", Date), f("start_date", "STARTDATE", Date),
		f("end_date", "ENDDATE", Date), f("is_actual", "ISACTUAL", Bool), f("is_active", "ISACTIVE", Bool),
	}
	hierarchy := []Field{
		f("id", "ID", Int64), f("object_id", "OBJECTID", Int64), f("parent_object_id", "PARENTOBJID", Int64),
		f("change_id", "CHANGEID", Int64), f("region_code", "REGIONCODE", Text), f("area_code", "AREACODE", Text),
		f("city_code", "CITYCODE", Text), f("place_code", "PLACECODE", Text), f("plan_code", "PLANCODE", Text),
		f("street_code", "STREETCODE", Text), f("path", "PATH", Text), f("update_date", "UPDATEDATE", Date),
		f("start_date", "STARTDATE", Date), f("end_date", "ENDDATE", Date), f("is_active", "ISACTIVE", Bool),
	}
	property := func(numberColumn, numberAttr string, extra ...Field) []Field {
		fields := []Field{
			f("id", "ID", Int64), f("object_id", "OBJECTID", Int64), f("object_guid", "OBJECTGUID", UUID),
			f("change_id", "CHANGEID", Int64), f(numberColumn, numberAttr, Text),
		}
		fields = append(fields, extra...)
		return append(fields,
			f("operation_type_id", "OPERTYPEID", Int), f("update_date", "UPDATEDATE", Date),
			f("start_date", "STARTDATE", Date), f("end_date", "ENDDATE", Date),
			f("is_actual", "ISACTUAL", Bool), f("is_active", "ISACTIVE", Bool),
		)
	}
	return []Family{
		family("address_objects", "AS_ADDR_OBJ_", "gar_address_objects", 1, []string{"OBJECT"}, objectFields),
		family("addr_object_params", "AS_ADDR_OBJ_PARAMS_", "gar_addr_object_params", 2, []string{"PARAM"}, commonParams),
		family("addr_object_divisions", "AS_ADDR_OBJ_DIVISION_", "gar_addr_object_divisions", 3, []string{"ITEM"}, []Field{
			f("id", "ID", Int64), f("parent_id", "PARENTID", Int64), f("child_id", "CHILDID", Int64), f("change_id", "CHANGEID", Int64),
		}),
		family("adm_hierarchy", "AS_ADM_HIERARCHY_", "gar_adm_hierarchy", 4, []string{"ITEM"}, hierarchy),
		family("mun_hierarchy", "AS_MUN_HIERARCHY_", "gar_mun_hierarchy", 5, []string{"ITEM"}, append(copyFields(hierarchy), f("oktmo", "OKTMO", Text))),
		family("houses", "AS_HOUSES_", "gar_houses", 6, []string{"HOUSE"}, property("house_num", "HOUSENUM",
			f("build_num", "BUILDNUM", Text), f("struc_num", "STRUCNUM", Text), f("house_type", "HOUSETYPE", Int),
			f("add_type1", "ADDTYPE1", Int), f("add_num1", "ADDNUM1", Text), f("add_type2", "ADDTYPE2", Int), f("add_num2", "ADDNUM2", Text))),
		family("house_params", "AS_HOUSES_PARAMS_", "gar_house_params", 7, []string{"PARAM"}, commonParams),
		family("apartments", "AS_APARTMENTS_", "gar_apartments", 8, []string{"APARTMENT"}, property("number", "NUMBER",
			f("apart_number", "APARTNUMBER", Text), f("apart_type", "APARTTYPE", Int))),
		family("apartment_params", "AS_APARTMENTS_PARAMS_", "gar_apartment_params", 9, []string{"PARAM"}, commonParams),
		family("carplaces", "AS_CARPLACES_", "gar_carplaces", 10, []string{"CARPLACE"}, property("number", "NUMBER")),
		family("carplace_params", "AS_CARPLACES_PARAMS_", "gar_carplace_params", 11, []string{"PARAM"}, commonParams),
		family("rooms", "AS_ROOMS_", "gar_rooms", 12, []string{"ROOM"}, property("number", "NUMBER", f("room_type", "ROOMTYPE", Int))),
		family("room_params", "AS_ROOMS_PARAMS_", "gar_room_params", 13, []string{"PARAM"}, commonParams),
		family("steads", "AS_STEADS_", "gar_steads", 14, []string{"STEAD"}, property("number", "NUMBER")),
		family("stead_params", "AS_STEADS_PARAMS_", "gar_stead_params", 15, []string{"PARAM"}, commonParams),
		family("change_history", "AS_CHANGE_HISTORY_", "gar_change_history", 16, []string{"CHANGEHISTORY", "ITEM"}, []Field{
			f("change_id", "CHANGEID", Int64), f("object_id", "OBJECTID", Int64), f("address_object_id", "ADROBJECTID", UUID),
			f("operation_type_id", "OPERTYPEID", Int), f("normative_doc_id", "NDOCID", Int64), f("change_date", "CHANGEDATE", Date),
		}),
		family("normative_docs", "AS_NORMATIVE_DOCS_", "gar_normative_docs", 17, []string{"NORMDOC", "DOCUMENT"}, []Field{
			f("id", "ID", Int64), f("name", "NAME", Text), f("number", "NUMBER", Text), f("document_date", "DATE", Date),
			f("document_type", "TYPE", Int), f("document_kind", "KIND", Int), f("update_date", "UPDATEDATE", Date),
			f("organization_name", "ORGNAME", Text), f("registration_number", "REGNUM", Text), f("registration_date", "REGDATE", Date),
			f("acceptance_date", "ACCDATE", Date), f("comment", "COMMENT", Text),
		}),
		family("reestr_objects", "AS_REESTR_OBJECTS_", "gar_reestr_objects", 18, []string{"REESTROBJECT", "OBJECT"}, []Field{
			f("object_id", "OBJECTID", Int64), f("object_guid", "OBJECTGUID", UUID), f("change_id", "CHANGEID", Int64),
			f("level_id", "LEVELID", Int), f("create_date", "CREATEDATE", Date), f("update_date", "UPDATEDATE", Date), f("is_active", "ISACTIVE", Bool),
		}),
	}
}

func family(key, prefix, table string, order int, elements []string, fields []Field) Family {
	accepted := make(map[string]struct{}, len(elements))
	for _, element := range elements {
		accepted[element] = struct{}{}
	}
	return Family{Key: key, Prefix: prefix, Table: table, Elements: accepted, Fields: copyFields(fields), Order: order}
}

func f(column, attr string, kind ValueKind) Field {
	return Field{Column: column, Attr: attr, Kind: kind}
}

func copyFields(fields []Field) []Field {
	return append([]Field(nil), fields...)
}
