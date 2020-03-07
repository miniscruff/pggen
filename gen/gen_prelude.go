package gen

import (
	"bufio"
	"os"
	"path/filepath"
	"text/template"
)

func (g *Generator) genPrelude() error {
	preludeName := filepath.Join(filepath.Dir(g.config.OutputFileName), "pggen_prelude.gen.go")
	outFile, err := os.OpenFile(preludeName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	out := bufio.NewWriter(outFile)
	defer out.Flush()

	type PreludeTmplCtx struct {
		Pkg string
	}
	tmplCtx := PreludeTmplCtx{
		Pkg: g.pkg,
	}
	return preludeTmpl.Execute(out, tmplCtx)
}

var preludeTmpl *template.Template = template.Must(template.New("prelude-tmpl").Parse(`
// Code generated by pggen. DO NOT EDIT

package {{ .Pkg }}

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
	uuid "github.com/satori/go.uuid"

	"github.com/opendoor-labs/pggen"
)

// PGClient wraps either a 'sql.DB' or a 'sql.Tx'. All pggen-generated
// database access methods for package {{ .Pkg }} are attached to it.
type PGClient struct {
	DB pggen.DBHandle
}

func genBulkInsert(
	table string,
	fields []string,
	nrecords int,
	pkeyName string,
) string {
	var ret strings.Builder

	ret.WriteString("INSERT INTO \"")
	ret.WriteString(table)
	ret.WriteString("\" (")
	for i, field := range fields {
		ret.WriteRune('"')
		ret.WriteString(field)
		ret.WriteRune('"')
		if i + 1 < len(fields) {
			ret.WriteRune(',')
		}
	}
	ret.WriteString(") VALUES ")

	for recNo := 0; recNo < nrecords; recNo++ {
		slots := make([]string, len(fields))[:0]
		for colNo := 1; colNo <= len(fields); colNo++ {
			slots = append(
				slots,
				fmt.Sprintf("$%d", (recNo * len(fields)) + colNo),
			)
		}
		ret.WriteString("	(")
		ret.WriteString(strings.Join(slots, ", "))
		if recNo < nrecords - 1 {
			ret.WriteString("),\n")
		} else {
			ret.WriteString(")\n")
		}
	}

	ret.WriteString(" RETURNING \"")
	ret.WriteString(pkeyName)
	ret.WriteRune('"')

	return ret.String()
}

func genUpdateStmt(
	table string,
	pgPkey string,
	fields []string,
	fieldMask pggen.FieldSet,
	pkeyName string,
) string {
	var ret strings.Builder

	ret.WriteString("UPDATE \"")
	ret.WriteString(table)
	ret.WriteString("\" SET ")

	lhs := make([]string, len(fields))[:0]
	rhs := make([]string, len(fields))[:0]
	argNo := 1
	for i, f := range fields {
		if fieldMask.Test(i) {
			lhs = append(lhs, f)
			rhs = append(rhs, fmt.Sprintf("$%d", argNo))
			argNo++
		}
	}

	if len(lhs) > 1 {
		ret.WriteRune('(')
		for i, f := range lhs {
			ret.WriteRune('"')
			ret.WriteString(f)
			ret.WriteRune('"')
			if i + 1 < len(lhs) {
				ret.WriteRune(',')
			}
		}
		ret.WriteRune(')')
	} else {
		ret.WriteRune('"')
		ret.WriteString(lhs[0])
		ret.WriteRune('"')
	}
	ret.WriteString(" = ")
	if len(rhs) > 1 {
		ret.WriteString(parenWrap(strings.Join(rhs, ", ")))
	} else {
		ret.WriteString(rhs[0])
	}
	ret.WriteString(" WHERE \"")
	ret.WriteString(pgPkey)
	ret.WriteString("\" = ")
	ret.WriteString(fmt.Sprintf("$%d", argNo))

	ret.WriteString(" RETURNING \"")
	ret.WriteString(pkeyName)
	ret.WriteRune('"')

	return ret.String()
}

func parenWrap(in string) string {
	return "(" + in + ")"
}

func convertNullString(s sql.NullString) *string {
	if s.Valid {
		return &s.String
	}
	return nil
}

func convertNullBool(b sql.NullBool) *bool {
	if b.Valid {
		return &b.Bool
	}
	return nil
}

// PggenPolyNullTime is shipped as sql.NullTime in go 1.13, but
// older versions of go don't have it yet, so we just roll it ourselves
// for compatibility.
type pggenNullTime struct {
	Time time.Time
	Valid bool
}
func (n *pggenNullTime) Scan(value interface{}) error {
	if value == nil {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	n.Valid = true

	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("scanning to NullTime: expected time.Time")
	}
	n.Time = t
	return nil
}
func (n pggenNullTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

func convertNullTime(t pggenNullTime) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

func convertNullFloat64(f sql.NullFloat64) *float64 {
	if f.Valid {
		return &f.Float64
	}
	return nil
}

func convertNullInt64(i sql.NullInt64) *int64 {
	if i.Valid {
		return &i.Int64
	}
	return nil
}

func convertNullUUID(u uuid.NullUUID) *uuid.UUID {
	if u.Valid {
		return &u.UUID
	}
	return nil
}
`))
