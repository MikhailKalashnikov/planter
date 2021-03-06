package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"sort"
	"strings"
    "os"
	_ "github.com/lib/pq" // postgres
	"github.com/pkg/errors"
)

// Queryer database/sql compatible query interface
type Queryer interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

// OpenDB opens database connection
func OpenDB(connStr string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	return conn, nil
}

// Column postgres columns
type Column struct {
	FieldOrdinal int
	Name         string
	Comment      sql.NullString
	DataType     string
	NotNull      bool
	IsPrimaryKey bool
	IsUnique bool
	DefVal sql.NullString
}

// ForeignKey foreign key
type ForeignKey struct {
	ConstraintName        string
	SourceTableName       string
	SourceTable           *Table
	TargetTableName       string
    ConstraintSchemaName  string
    SourceSchemaName      string
}

// Table postgres table
type Table struct {
	Schema      string
	Name        string
	Comment     sql.NullString
	AutoGenPk   bool
	Columns     []*Column
	ForeingKeys []*ForeignKey
}

// IsCompositePK check if table is composite pk
func (t *Table) IsCompositePK() bool {
	cnt := 0
	for _, c := range t.Columns {
		if c.IsPrimaryKey {
			cnt++
		}
		if cnt >= 2 {
			return true
		}
	}
	return false
}

func stripCommentSuffix(s string) string {
	if tok := strings.SplitN(s, "\t", 2); len(tok) == 2 {
		return tok[0]
	}
	return s
}

// FindTableByName find table by name
func FindTableByName(tbls []*Table, name string) (*Table, bool) {
	for _, tbl := range tbls {
		if tbl.Name == name {
			return tbl, true
		}
	}
	return nil, false
}

// FindColumnByName find table by name
func FindColumnByName(tbls []*Table, tableName, colName string) (*Column, bool) {
	for _, tbl := range tbls {
		if tbl.Name == tableName {
			for _, col := range tbl.Columns {
				if col.Name == colName {
					return col, true
				}
			}
		}
	}
	return nil, false
}

// LoadColumnDef load Postgres column definition
func LoadColumnDef(db Queryer, schema, table string) ([]*Column, error) {
	colDefs, err := db.Query(columDefSQL, schema, table)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def")
	}
	var cols []*Column
	for colDefs.Next() {
		var c Column
		err := colDefs.Scan(
			&c.FieldOrdinal,
			&c.Name,
			&c.Comment,
			&c.DataType,
			&c.NotNull,
			&c.IsPrimaryKey,
			&c.IsUnique,
			&c.DefVal,
		)
		c.Comment.String = stripCommentSuffix(c.Comment.String)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		cols = append(cols, &c)
	}
	return cols, nil
}

// LoadForeignKeyDef load Postgres fk definition
func LoadForeignKeyDef(db Queryer, schema string, tbls []*Table, tbl *Table) ([]*ForeignKey, error) {
	fkDefs, err := db.Query(fkDefSQL, schema, tbl.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load fk def")
	}
	var fks []*ForeignKey
	for fkDefs.Next() {
		fk := ForeignKey{
			SourceTableName: tbl.Name,
			SourceTable:     tbl,
		}
		err := fkDefs.Scan(
			&fk.TargetTableName,
			&fk.ConstraintName,
			&fk.ConstraintSchemaName,
            		&fk.SourceSchemaName,
		)
		if err != nil {
			return nil, err
		}
		fks = append(fks, &fk)
	}
// 	for _, fk := range fks {
// 		targetTbl, found := FindTableByName(tbls, fk.TargetTableName)
// 		if !found {
// 			return nil, errors.Errorf("%s not found", fk.TargetTableName)
// 		}
// 		fk.TargetTable = targetTbl
// 		targetCol, found := FindColumnByName(tbls, fk.TargetTableName, fk.TargetColName)
// 		if !found {
// 			return nil, errors.Errorf("%s.%s not found", fk.TargetTableName, fk.TargetColName)
// 		}
// 		fk.TargetColumn = targetCol
// 		sourceCol, found := FindColumnByName(tbls, fk.SourceTableName, fk.SourceColName)
// 		if !found {
// 			return nil, errors.Errorf("%s.%s not found", fk.SourceTableName, fk.SourceColName)
// 		}
// 		fk.SourceColumn = sourceCol
// 	}
	return fks, nil
}

// LoadTableDefForSchemas load Postgres table definition
func LoadTableDefForSchemas(db Queryer, schemas []string, skipFlags string) ([]*Table, error) {
    var tbls []*Table
	for _, schema := range schemas {
		tbls2, err := LoadTableDef(db, schema, skipFlags)
		tbls = append(tbls, tbls2...)
		if err != nil {
            return tbls, err
        }

	}
	return tbls, nil
}

// LoadTableDef load Postgres table definition
func LoadTableDef(db Queryer, schema string, skipFlags string) ([]*Table, error) {
    fmt.Fprintln(os.Stdout, "Load schema: " + schema)
	tbDefs, err := db.Query(tableDefSQL, schema)
	var tbls []*Table
	if err != nil {
		return nil, errors.Wrap(err, "failed to load table def")
	}
	for tbDefs.Next() {
		t := &Table{Schema: schema}
		err := tbDefs.Scan(
			&t.Name,
			&t.Comment,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		fmt.Fprintln(os.Stdout, "Load table: " + schema + "." + t.Name)
		cols, err := LoadColumnDef(db, schema, t.Name)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get columns of %s", t.Name))
		}
		t.Columns = cols
		tbls = append(tbls, t)
	}
	if !strings.Contains(skipFlags, "f") {
	    for _, tbl := range tbls {
    		fks, err := LoadForeignKeyDef(db, schema, tbls, tbl)
    		if err != nil {
    			return nil, errors.Wrap(err, fmt.Sprintf("failed to get fks of %s", tbl.Name))
    		}
    		tbl.ForeingKeys = fks
    	}
	}
	return tbls, nil
}

// TableToUMLEntry table entry
func TableToUMLEntry(tbls []*Table) ([]byte, error) {
	tpl, err := template.New("entry").Parse(entryTmpl)
	if err != nil {
		return nil, err
	}
	var src []byte
	for _, tbl := range tbls {
		buf := new(bytes.Buffer)
		if err := tpl.Execute(buf, tbl); err != nil {
			return nil, errors.Wrapf(err, "failed to execute template: %s", tbl.Name)
		}
		src = append(src, buf.Bytes()...)
	}
	return src, nil
}

// TableToUMLTable table entry
func TableToUMLTable(tbl *Table) ([]byte, error) {
	tpl, err := template.New("table").Parse(tableTmpl)
	if err != nil {
		return nil, err
	}
    buf := new(bytes.Buffer)
    if err := tpl.Execute(buf, tbl); err != nil {
        return nil, errors.Wrapf(err, "failed to execute template: %s", tbl.Name)
    }
	return buf.Bytes(), nil
}

// TableToRSTTable table entry
func TableToRSTTable(tbl *Table) ([]byte, error) {
	tpl, err := template.New("rsttable").Parse(rstTableTmpl)
	if err != nil {
		return nil, err
	}
    buf := new(bytes.Buffer)
    if err := tpl.Execute(buf, tbl); err != nil {
        return nil, errors.Wrapf(err, "failed to execute template: %s", tbl.Name)
    }
	return buf.Bytes(), nil
}

// ForeignKeyToUMLRelation relation
func ForeignKeyToUMLRelation(tbls []*Table) ([]byte, error) {
	tpl, err := template.New("relation").Parse(relationTmpl)
	if err != nil {
		return nil, err
	}
	var src []byte
	for _, tbl := range tbls {
		for _, fk := range tbl.ForeingKeys {
			buf := new(bytes.Buffer)
			if err := tpl.Execute(buf, fk); err != nil {
				return nil, errors.Wrapf(err, "failed to execute template: %s", fk.ConstraintName)
			}
			src = append(src, buf.Bytes()...)
		}
	}
	return src, nil
}

// ForeignKeyToUMLRelation2 relation
func ForeignKeyToUMLRelation2(tbl *Table) ([]byte, []byte, error) {
    fmt.Fprintln(os.Stdout, "ForeignKeyToUMLRelation2: " + tbl.Name)
	tpl, err := template.New("relation").Parse(relationTmpl)
	if err != nil {
		return nil, nil, err
	}
	var schema_src1 []byte
	var global_src2 []byte
	for _, fk := range tbl.ForeingKeys {
        buf := new(bytes.Buffer)
        if err := tpl.Execute(buf, fk); err != nil {
            return nil, nil, errors.Wrapf(err, "failed to execute template: %s", fk.ConstraintName)
        }
        if fk.ConstraintSchemaName != fk.SourceSchemaName {
            global_src2 = append(global_src2, buf.Bytes()...)
        } else {
            schema_src1 = append(schema_src1, buf.Bytes()...)
        }
    }
	return schema_src1, global_src2, nil
}

func contains(v string, l []string) bool {
	i := sort.SearchStrings(l, v)
	if i < len(l) && l[i] == v {
		return true
	}
	return false
}

// FilterTables filter tables
func FilterTables(match bool, tbls []*Table, tblNames []string) []*Table {
	sort.Strings(tblNames)

	var target []*Table
	for _, tbl := range tbls {
		if contains(tbl.Name, tblNames) == match {
			var fks []*ForeignKey
			for _, fk := range tbl.ForeingKeys {
				if contains(fk.TargetTableName, tblNames) == match {
					fks = append(fks, fk)
				}
			}
			tbl.ForeingKeys = fks
			target = append(target, tbl)
		}
	}
	return target
}

// FilterTableSuffix filter tables by suffix
func FilterTableSuffix(tbls []*Table, xTblNameSuffix string) []*Table {
	var target []*Table
	for _, tbl := range tbls {
		if strings.HasSuffix(tbl.Name, xTblNameSuffix) == false {
			var fks []*ForeignKey
			for _, fk := range tbl.ForeingKeys {
				if strings.HasSuffix(fk.TargetTableName, xTblNameSuffix) == false {
					fks = append(fks, fk)
				}
			}
			tbl.ForeingKeys = fks
			target = append(target, tbl)
		}
	}
	return target
}


func printtable(tbls []*Table) {
	for _, tbl := range tbls {
	    fmt.Fprintln(os.Stdout, "printtable : " + tbl.Name)

        for _, fk := range tbl.ForeingKeys {
            fmt.Fprintln(os.Stdout, "        : " + fk.ConstraintName)
        }
	}
}
