package main

import (
	"io"
	"log"
	"os"
    "fmt"
    "path/filepath"
    "strings"
	"github.com/alecthomas/kingpin"
)

var (
	connStr = kingpin.Arg(
		"conn", "PostgreSQL connection string in URL format").Required().String()
	schemas = kingpin.Flag(
		"schema", "PostgreSQL schemas name").Default("public").Short('s').Strings()
	outFile     = kingpin.Flag("output", "output file path").Short('o').String()
    outDir     = kingpin.Flag("output_dir", "output dir path").Short('p').String()
    dbName     = kingpin.Flag("dbname", "dbName for UML").Short('d').String()
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

    if *outDir != "" {
        static_file_erd(*outDir);
        static_file_legend(*outDir);

        var main_src []byte
        main_src = append([]byte("@startuml\n"))
        main_src = append(main_src, []byte("skinparam monochrome true\n")...)
        main_src = append(main_src, []byte("!ifndef ERD_INCL\n")...)
        main_src = append(main_src, []byte("!include erd.iuml\n")...)
        main_src = append(main_src, []byte("!endif\n")...)
        main_src = append(main_src, []byte("package " + *dbName + " <<Database>> {\n")...)

        var main_rel_src []byte
        main_rel_src = append([]byte("\n"))

        var rst_src []byte
        rst_src = append([]byte("\n"))

        for _, schema := range *schemas {
            fmt.Fprintln(os.Stdout, "Extract schema: " + schema)
            var schemaDir string
            schemaDir = filepath.Join(*outDir, schema)
            os.Mkdir(schemaDir, 0777);
    		ts, err := LoadTableDef(db, schema, *skipFlags)
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

            var schema_src []byte
            var schema_rel_src []byte
            schema_src = append([]byte("@startuml\n"))
            schema_rel_src = append([]byte("\n"))
            schema_src = append(schema_src, []byte("skinparam monochrome true\n")...)
            schema_src = append(schema_src, []byte("!ifndef ERD_INCL\n")...)
            schema_src = append(schema_src, []byte("!include ../erd.iuml\n")...)
            schema_src = append(schema_src, []byte("!endif\n")...)
            schema_src = append(schema_src, []byte("package " + schema + " <<Frame>> {\n")...)

            rst_src = append(rst_src, []byte(strings.ToUpper(schema) + "\n")...)
            rst_src = append(rst_src, []byte("----\n")...)

            for _, tbl := range tbls {
                schema_src = append(schema_src, []byte("!include " + tbl.Name + ".puml\n")...)
                umlTable, err := TableToUMLTable(tbl)
                if err != nil {
                    log.Fatal(err)
                }

                var outFileTbl string;
                outFileTbl = filepath.Join(schemaDir, tbl.Name + ".puml")

                if err := write_to_file(outFileTbl, umlTable); err != nil {
                    log.Fatal(err)
                }

                schema_rel1, global_rel2, err := ForeignKeyToUMLRelation2(tbl)
                if err != nil {
                    log.Fatal(err)
                }

                schema_rel_src = append(schema_rel_src, schema_rel1...)
                main_rel_src = append(main_rel_src, global_rel2...)

                rstTable, err := TableToRSTTable(tbl)
                if err != nil {
                    log.Fatal(err)
                }
                rst_src = append(rst_src, rstTable...)
            }

            schema_src = append(schema_src, schema_rel_src...)

            schema_src = append(schema_src, []byte("}\n")...)
            schema_src = append(schema_src, []byte("!ifndef LEGEND_INCL\n")...)
            schema_src = append(schema_src, []byte("!include ../legend.iuml\n")...)
            schema_src = append(schema_src, []byte("!endif\n")...)
            schema_src = append(schema_src, []byte("@enduml\n")...)

            var outFileSchema string;
            outFileSchema = filepath.Join(schemaDir, "_schema.puml")
            if err := write_to_file(outFileSchema, schema_src); err != nil {
                log.Fatal(err)
            }

            main_src = append(main_src, []byte("!include " + schema + "/_schema.puml\n")...)
            rst_src = append(rst_src, []byte("\n\n")...)
    	}

        main_src = append(main_src, main_rel_src...)

        main_src = append(main_src, []byte("}\n")...)
        main_src = append(main_src, []byte("!ifndef LEGEND_INCL\n")...)
        main_src = append(main_src, []byte("!include legend.iuml\n")...)
        main_src = append(main_src, []byte("!endif\n")...)
        main_src = append(main_src, []byte("@enduml\n")...)

        var outFileMain string;
        outFileMain = filepath.Join(*outDir, "sql-db-" + *dbName + "-er.puml")
        if err := write_to_file(outFileMain, main_src); err != nil {
            log.Fatal(err)
        }

        var outFileRST string;
        outFileRST = filepath.Join(*outDir, "description.rst")
        if err := write_to_file(outFileRST, rst_src); err != nil {
            log.Fatal(err)
        }


    } else {
        ts, err := LoadTableDefForSchemas(db, *schemas, *skipFlags)
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
}


func static_file_erd(outDir string) (error) {
    var src []byte
    src = append([]byte(
`!define ERD_INCL
!define table(x) class x << (T,#FFAAAA) >>
!define pk(x) <u>x</u>
hide methods
hide stereotypes
`))
	var outFile string;
	outFile = filepath.Join(outDir, "erd.iuml")

	return write_to_file(outFile, src)
}

func static_file_legend(outDir string) (error) {
    var src []byte
    src = append([]byte(
`!define LEGEND_INCL
legend right
    <b>NN</b> - NOT NULL
    <b>UN</b> - UNIQUE
    <b>field=value</b> - DEFAULT value
    <b><u>field</u></b> - Primary Key
endlegend
`))

	var outFile string;
	outFile = filepath.Join(outDir, "legend.iuml")
	return write_to_file(outFile, src)
}



func write_to_file(outFile string, src []byte) (error) {
    var err error
	var out io.Writer
	out, err = os.Create(outFile)
    if err != nil {
        log.Fatalf("failed to create output file %s: %s", outFile, err)
    }
	if _, err := out.Write(src); err != nil {
		log.Fatal(err)
	}
	return err
}