package seatax

import (
	"sync"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm/schema"
)

func TestUndoLogTableName(t *testing.T) {
	if got := (UndoLog{}).TableName(); got != "undo_log" {
		t.Fatalf("TableName() = %q, want undo_log", got)
	}
}

func TestUndoLogSchema(t *testing.T) {
	s, err := schema.Parse(&UndoLog{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse UndoLog schema error: %v", err)
	}
	if s.Table != "undo_log" {
		t.Fatalf("schema table = %q, want undo_log", s.Table)
	}

	id := s.LookUpField("ID")
	if id == nil {
		t.Fatal("field ID not found in UndoLog schema")
	}
	if !id.PrimaryKey || !id.AutoIncrement {
		t.Fatalf("ID primaryKey=%v autoIncrement=%v, want both true", id.PrimaryKey, id.AutoIncrement)
	}

	// 字符串列统一使用 text（PostgreSQL 的 text / MySQL 的 longtext），避免长度溢出。
	const textType = schema.DataType("text")
	for _, name := range []string{"Xid", "Context", "Ext"} {
		f := s.LookUpField(name)
		if f == nil {
			t.Fatalf("field %s not found in UndoLog schema", name)
		}
		if f.DataType != textType {
			t.Fatalf("field %s dataType = %v, want %v", name, f.DataType, textType)
		}
	}

	// 回滚镜像列由 GORM 按方言渲染为 bytea（PostgreSQL）/ longblob（MySQL）。
	rollback := s.LookUpField("RollbackInfo")
	if rollback == nil {
		t.Fatal("field RollbackInfo not found in UndoLog schema")
	}
	if rollback.DataType != schema.Bytes {
		t.Fatalf("field RollbackInfo dataType = %v, want %v", rollback.DataType, schema.Bytes)
	}

	parsedIndexes := s.ParseIndexes()
	indexes := make(map[string]*schema.Index, len(parsedIndexes))
	for _, idx := range parsedIndexes {
		indexes[idx.Name] = idx
	}

	unique, ok := indexes["ux_undo_log"]
	if !ok {
		t.Fatalf("unique index ux_undo_log not found, got indexes: %v", indexNames(parsedIndexes))
	}
	if unique.Class != "UNIQUE" {
		t.Fatalf("index ux_undo_log class = %q, want UNIQUE", unique.Class)
	}
	if got := indexFields(unique.Fields); len(got) != 2 || got[0] != "xid" || got[1] != "branch_id" {
		t.Fatalf("index ux_undo_log fields = %v, want [xid branch_id]", got)
	}

	created, ok := indexes["ix_log_created"]
	if !ok {
		t.Fatalf("index ix_log_created not found, got indexes: %v", indexNames(parsedIndexes))
	}
	if got := indexFields(created.Fields); len(got) != 1 || got[0] != "log_created" {
		t.Fatalf("index ix_log_created fields = %v, want [log_created]", got)
	}
}

func TestMigrateUndoLogNilDB(t *testing.T) {
	if err := MigrateUndoLog(nil); err == nil {
		t.Fatal("MigrateUndoLog(nil) returns nil error, want an error")
	}
}

func TestUndoLogDialectTypes(t *testing.T) {
	s, err := schema.Parse(&UndoLog{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse UndoLog schema error: %v", err)
	}

	pg := postgres.Dialector{Config: &postgres.Config{DriverName: "pgx"}}
	my := mysql.Dialector{Config: &mysql.Config{
		DriverName: "mysql",
		DSN:        "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true",
	}}

	// PostgreSQL 是本次迁移的目标方言：自增主键为 bigserial、字符串为 text、镜像为 bytea。
	pgWant := map[string]string{
		"ID":           "bigserial",
		"Xid":          "text",
		"Context":      "text",
		"Ext":          "text",
		"RollbackInfo": "bytea",
		"LogStatus":    "integer",
	}
	// MySQL 与 undo_log.sql 同理：bigint auto_increment、text、longblob、int。
	myWant := map[string]string{
		"ID":           "bigint AUTO_INCREMENT",
		"Xid":          "text",
		"Context":      "text",
		"Ext":          "text",
		"RollbackInfo": "longblob",
		"LogStatus":    "int",
	}

	for name, want := range pgWant {
		f := s.LookUpField(name)
		if f == nil {
			t.Fatalf("field %s not found in UndoLog schema", name)
		}
		if got := pg.DataTypeOf(f); got != want {
			t.Fatalf("postgres field %s dataType = %q, want %q", name, got, want)
		}
	}
	for name, want := range myWant {
		f := s.LookUpField(name)
		if f == nil {
			t.Fatalf("field %s not found in UndoLog schema", name)
		}
		if got := my.DataTypeOf(f); got != want {
			t.Fatalf("mysql field %s dataType = %q, want %q", name, got, want)
		}
	}
}

func indexFields(fields []schema.IndexOption) []string {
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.DBName)
	}
	return names
}

func indexNames(indexes []*schema.Index) []string {
	names := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		names = append(names, idx.Name)
	}
	return names
}
