//go:build helpers

package main

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

const outputFile = "queries.go"

var (
	queryVarTmpl = must(template.New("queryVarTmpl").Parse(`
	//go:embed {{.FileName}}
	{{.VarName}} string{{ "" -}}
`))
	prepareFuncsTmpl = must(template.New("prepareFuncsTmpl").Parse(`
// {{.FuncName}} prepares a query for execution by replacing named parameters with positional parameters.
{{if .Args}}// Required NamedArg{{end}}
{{range .Args}}//   - {{.}}
{{end -}}
func {{.FuncName}}(namedArg ...sql.NamedArg) (string, []interface{}) {
	return Prepare({{.VarName}}, namedArg)
}
`))
	helperTmpl = must(template.New("helperTmpl").Parse(`
{{- ""}}// Code generated by gen_helpers.go DO NOT EDIT.
// Source: *.sql

// This file is a generated from {{.GoSourceFile}}.
//go:build !helpers

package queries

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

//go:generate go run -tags helpers {{.GoSourceFile}}
var ({{.Queries}}
)

func Prepare(query string, args []sql.NamedArg) (string, []interface{}) {
	vals := make([]interface{}, 0)
	count := 1
	for _, arg := range args {
		param := fmt.Sprintf("@%s", arg.Name)
		if strings.Contains(query, param) {
			pos := fmt.Sprintf("$%d", count)
			query = strings.Replace(query, param, pos, -1)
			vals = append(vals, arg.Value)
			count++
		}
	}
	log.Debug().Str("migration-query", query).Interface("args", vals).Msg("prepared query")
	return query, vals
}
{{ .PrepareFuncs -}}
`))
	matchUnderscore = regexp.MustCompile("_([a-zA-Z])")
	matchQueryParam = regexp.MustCompile("@([a-zA-Z0-9_]+)")
)

type SqlQueryData struct {
	FileName, VarName, FuncName string
	Args                        []string
}

type HelperData struct {
	_Queries, _PrepareFuncs *bytes.Buffer
	GoSourceFile            string
}

func main() {
	data := &HelperData{
		_Queries:      &bytes.Buffer{},
		_PrepareFuncs: &bytes.Buffer{},
		GoSourceFile:  "gen_queries.go",
	}
	for _, query := range getSqlFiles() {
		if err := queryVarTmpl.Execute(data._Queries, query); err != nil {
			log.Fatalf("queryVarTmpl execution failed: %s", err)
		}
		if err := prepareFuncsTmpl.Execute(data._PrepareFuncs, query); err != nil {
			log.Fatalf(" prepareFuncsTmpl execution failed: %s", err)
		}
	}
	file := must(os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644))
	if err := helperTmpl.Execute(file, data); err != nil {
		log.Fatalf(" helperTmpl execution failed: %s", err)
	}
}

func (d *HelperData) Queries() string {
	return d._Queries.String()
}
func (d *HelperData) PrepareFuncs() string {
	return d._PrepareFuncs.String()
}

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func getSqlFiles() []*SqlQueryData {
	files, err := os.ReadDir(".")
	if err != nil {
		log.Fatalf("failed to read current directory: %w", err)
	}
	queries := make([]*SqlQueryData, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			query := &SqlQueryData{
				Args:     getQueryParameters(file.Name()),
				FileName: file.Name(),
				VarName:  strings.Replace(file.Name(), ".sql", "", -1),
			}
			for _, matches := range matchUnderscore.FindAllStringSubmatch(file.Name(), -1) {
				query.VarName = strings.Replace(query.VarName, matches[0], strings.ToUpper(matches[1]), -1)
			}
			query.FuncName = strings.ToUpper(string(query.VarName[0])) + query.VarName[1:]
			queries = append(queries, query)
		}
	}
	return queries
}

func getQueryParameters(fileName string) []string {
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("failed to read %s files in current directory: %w", fileName, err)
	}
	uniqParams := make(map[string]bool)
	params := make([]string, 0)
	for _, matches := range matchQueryParam.FindAllStringSubmatch(string(data), -1) {
		uniqParams[matches[1]] = true
	}
	for param := range uniqParams {
		params = append(params, param)
	}
	sort.Strings(params)
	return params
}