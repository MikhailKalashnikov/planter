package main

import (
	"io"
	"log"
	"os"
//     "fmt"
	"github.com/alecthomas/kingpin"
)

var (
	connStr = kingpin.Arg(
		"conn", "PostgreSQL connection string in URL format").Required().String()
	schema = kingpin.Flag(
		"schema", "PostgreSQL schemas name").Default("public").Short('s').Strings()
	outFile     = kingpin.Flag("output", "output file path").Short('o').String()
	targetTbls  = kingpin.Flag("table", "target tables").Short('t').Strings()
	xTargetTbls = kingpin.Flag("exclude", "target tables").Short('x').Strings()
	xTblNameSuffix = kingpin.Flag("exclude_suffix", "exclude suffix").Short('f').String()
	skipFlags   = kingpin.Flag("skip_flags", "exclude suffix").Short('q').String()
)

func main() {
	kingpin.Parse()

	db, err := OpenDB(*connStr)
	if err != nil {
		log.Fatal(err)
	}

	ts, err := LoadTableDefForSchemas(db, *schema, *skipFlags)
	if err != nil {
		log.Fatal(err)
	}

	var tbls []*Table
	if len(*targetTbls) != 0 {
		tbls = FilterTables(true, ts, *targetTbls)
	} else {
		tbls = ts
	}
	if len(*xTargetTbls) != 0 {
		tbls = FilterTables(false, tbls, *xTargetTbls)
	}
	if xTblNameSuffix != nil && len(*xTblNameSuffix) > 0 {
        tbls = FilterTableSuffix(tbls, *xTblNameSuffix)
    }
	entry, err := TableToUMLEntry(tbls)
	if err != nil {
		log.Fatal(err)
	}
	rel, err := ForeignKeyToUMLRelation(tbls)
	if err != nil {
		log.Fatal(err)
	}
	var src []byte
	src = append([]byte("@startuml\n"), entry...)
	src = append(src, rel...)
	src = append(src, []byte("@enduml\n")...)

	var out io.Writer
	if *outFile != "" {
		out, err = os.Create(*outFile)
		if err != nil {
			log.Fatalf("failed to create output file %s: %s", *outFile, err)
		}
	} else {
		out = os.Stdout
	}
	if _, err := out.Write(src); err != nil {
		log.Fatal(err)
	}
}
