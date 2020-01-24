package gen

import (
	"fmt"
	"io"
	"strings"
	"text/template"
)

// Generate code for all of the tables
func (g *Generator) genTables(into io.Writer, tables []tableConfig) error {
	if len(tables) > 0 {
		g.infof("	generating %d tables\n", len(tables))
	} else {
		return nil
	}

	g.imports[`"database/sql"`] = true
	g.imports[`"context"`] = true
	g.imports[`"fmt"`] = true
	g.imports[`"math"`] = true
	g.imports[`"github.com/lib/pq"`] = true
	g.imports[`"github.com/willf/bitset"`] = true

	tableConfigs := map[string]*tableConfig{}
	for i, table := range tables {
		tableConfigs[table.Name] = &tables[i]
	}

	for _, table := range tables {
		err := g.genTable(into, tableConfigs, &table)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) genTable(
	into io.Writer,
	// A mapping from table names to table configs
	tableConfigs map[string]*tableConfig,
	table *tableConfig,
) (err error) {
	g.infof("		generating table '%s'\n", table.Name)
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"while generating table '%s': %s", table.Name, err.Error())
		}
	}()

	meta, err := g.tableMeta(table.Name)
	if err != nil {
		return
	}

	// Filter out all the references from tables that are not mentioned in the TOML.
	// We only want to generate code about the part of the database schema that we have
	// been explicitly asked to generate code for.
	kept := 0
	for _, ref := range meta.References {
		if _, inMap := tableConfigs[ref.PgPointsFrom]; inMap {
			meta.References[kept] = ref
			kept++
		}

		if len(ref.PointsFromFields) != 1 {
			err = fmt.Errorf("multi-column foreign keys not supported")
			return
		}
	}
	meta.References = meta.References[:kept]

	if meta.PkeyCol == nil {
		err = fmt.Errorf("no primary key for table")
		return
	}

	// Emit the type seperately to prevent double defintions
	var tableType strings.Builder
	err = tableTypeTmpl.Execute(&tableType, meta)
	if err != nil {
		return
	}
	var tableSig strings.Builder
	err = tableTypeFieldSigTmpl.Execute(&tableSig, meta)
	if err != nil {
		return
	}
	g.types.emitType(meta.GoName, tableSig.String(), tableType.String())

	return tableShimTmpl.Execute(into, meta)
}

var tableTypeFieldSigTmpl *template.Template = template.Must(template.New("table-type-field-sig-tmpl").Parse(`
{{- range .Cols }}
{{- if .Nullable }}
{{ .GoName }} {{ .TypeInfo.NullName }}
{{- else }}
{{ .GoName }} {{ .TypeInfo.Name }}
{{- end }}
{{- end }}
`))

var tableTypeTmpl *template.Template = template.Must(template.New("table-type-tmpl").Parse(`
type {{ .GoName }} struct {
	{{- range .Cols }}
	{{- if .Nullable }}
	{{ .GoName }} {{ .TypeInfo.NullName }}
	{{- else }}
	{{ .GoName }} {{ .TypeInfo.Name }}
	{{- end }} ` +
	"`" + `gorm:"column:{{ .PgName }}"
	{{- if .IsPrimary }} gorm:"is_primary" {{- end }}` +
	"`" + `
	{{- end }}
	{{- range .References }}
	{{ .PluralGoPointsFrom }} []{{ .GoPointsFrom }}
	{{- end }}
}
func (r *{{ .GoName }}) Scan(rs *sql.Rows) error {
	return rs.Scan(
		{{- range .Cols }}
		{{ call .TypeInfo.SqlReceiver (printf "r.%s" .GoName) }},
		{{- end }}
	)
}
`))

var tableShimTmpl *template.Template = template.Must(template.New("table-shim-tmpl").Parse(`

func (p *PGClient) Get{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) ({{ .GoName }}, error) {
	values, err := p.List{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id})
	if err != nil {
		return {{ .GoName }}{}, err
	}

	// List{{ .GoName }} always returns the same number of records as were
	// requested, so this is safe.
	return values[0], err
}

func (p *PGClient) List{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) ([]{{ .GoName }}, error) {
	rows, err := p.DB.QueryContext(
		ctx,
		"SELECT * FROM \"{{ .PgName }}\" WHERE {{ .PkeyCol.PgName }} = ANY($1)",
		pq.Array(ids),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := make([]{{ .GoName }}, len(ids))[:0]
	for rows.Next() {
		var value {{ .GoName }}
		err = value.Scan(rows)
		if err != nil {
			return nil, err
		}
		ret = append(ret, value)
	}

	if len(ret) != len(ids) {
		return nil, fmt.Errorf(
			"List{{ .GoName }}: asked for %d records, found %d",
			len(ids),
			len(ret),
		)
	}

	return ret, nil
}

// Insert a {{ .GoName }} into the database. Returns the primary
// key of the inserted row.
func (p *PGClient) Insert{{ .GoName }}(
	ctx context.Context,
	value {{ .GoName }},
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var ids []{{ .PkeyCol.TypeInfo.Name }}
	ids, err = p.BulkInsert{{ .GoName }}(ctx, []{{ .GoName }}{value})
	if err != nil {
		return
	}

	if len(ids) != 1 {
		err = fmt.Errorf("inserting a {{ .GoName }}: %d ids (expected 1)", len(ids))
		return
	}

	ret = ids[0]
	return
}
// Insert a list of {{ .GoName }}. Returns a list of the primary keys of
// the inserted rows.
func (p *PGClient) BulkInsert{{ .GoName }}(
	ctx context.Context,
	values []{{ .GoName }},
) ([]{{ .PkeyCol.TypeInfo.Name }}, error) {
	var fields []string = []string{
		{{- range .Cols }}
		{{- if (not .IsPrimary) }}
		"{{ .PgName }}",
		{{- end }}
		{{- end }}
	}

	args := make([]interface{}, {{ len .Cols }} * len(values))[:0]
	for _, v := range values {
		{{- range .Cols }}
		{{- if (not .IsPrimary) }}
		args = append(args, v.{{ .GoName }})
		{{- end }}
		{{- end }}
	}

	bulkInsertQuery := genBulkInsert(
		"{{ .PgName }}",
		fields,
		len(values),
		"{{ .PkeyCol.PgName }}",
	)

	rows, err := p.DB.QueryContext(ctx, bulkInsertQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]{{ .PkeyCol.TypeInfo.Name }}, len(values))[:0]
	for rows.Next() {
		var id {{ .PkeyCol.TypeInfo.Name }}
		err = rows.Scan({{ call .PkeyCol.TypeInfo.SqlReceiver "id" }})
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// bit indicies for 'fieldMask' parameters
const (
	{{- range $i, $c := .Cols }}
	{{ $.GoName }}{{ $c.GoName }}FieldIndex uint = {{ $i }}
	{{- end }}
)

// A bitset saying that all fields in {{ .GoName }} should be updated.
// For use as a 'fieldMask' parameter
var {{ .GoName }}AllFields *bitset.BitSet = func() *bitset.BitSet {
	ret := bitset.New({{ len .Cols }})
	var i uint
	for i = 0; i < uint({{ len .Cols }}); i++ {
		ret.Set(i)
	}
	return ret
}()

// Update a {{ .GoName }}. 'value' must at the least have
// a primary key set. The 'fieldMask' bitset indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *PGClient) Update{{ .GoName }}(
	ctx context.Context,
	value {{ .GoName }},
	fieldMask *bitset.BitSet,
) (ret {{ .PkeyCol.TypeInfo.Name }}, err error) {
	var fields []string = []string{
		{{- range .Cols }}
		"{{ .PgName }}",
		{{- end }}
	}

	if !fieldMask.Test({{ .GoName }}{{ .PkeyCol.GoName }}FieldIndex) {
		err = fmt.Errorf("primary key required for updates to '{{ .PgName }}'")
		return
	}

	updateStmt := genUpdateStmt(
		"{{ .PgName }}",
		"{{ .PkeyCol.PgName }}",
		fields,
		fieldMask,
		"{{ .PkeyCol.PgName }}",
	)

	args := make([]interface{}, {{ len .Cols }})[:0]

	{{- range .Cols }}
	if fieldMask.Test({{ $.GoName }}{{ .GoName }}FieldIndex) {
		args = append(args, value.{{ .GoName }})
	}
	{{- end }}

	// add the primary key arg for the WHERE condition
	args = append(args, value.{{ .PkeyCol.GoName }})

	var id {{ .PkeyCol.TypeInfo.Name }}
	err = p.DB.QueryRowContext(ctx, updateStmt, args...).
                Scan({{ call .PkeyCol.TypeInfo.SqlReceiver "id" }})
	if err != nil {
		return
	}

	return id, nil
}

func (p *PGClient) Delete{{ .GoName }}(
	ctx context.Context,
	id {{ .PkeyCol.TypeInfo.Name }},
) error {
	return p.BulkDelete{{ .GoName }}(ctx, []{{ .PkeyCol.TypeInfo.Name }}{id})
}

func (p *PGClient) BulkDelete{{ .GoName }}(
	ctx context.Context,
	ids []{{ .PkeyCol.TypeInfo.Name }},
) error {
	res, err := p.DB.ExecContext(
		ctx,
		"DELETE FROM \"{{ .PgName }}\" WHERE {{ .PkeyCol.PgName }} = ANY($1)",
		pq.Array(ids),
	)
	if err != nil {
		return err
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if nrows != int64(len(ids)) {
		return fmt.Errorf(
			"BulkDelete{{ .GoName }}: %d rows deleted, expected %d",
			nrows,
			len(ids),
		)
	}

	return err
}

func (p *PGClient) {{ .GoName }}FillAll(
	ctx context.Context,
	rec *{{ .GoName }},
) error {
	return p.{{ .GoName }}FillToDepth(ctx, []*{{ .GoName }}{rec}, math.MaxInt64)
}

func (p *PGClient) {{ .GoName }}FillToDepth(
	ctx context.Context,
	recs []*{{ .GoName }},
	maxDepth int64,
) (err error) {
	if maxDepth <= 0 {
		return
	}
{{- range .References }}

	// Fill in the {{ .PluralGoPointsFrom }}
	err = p.{{ $.GoName }}Fill{{ .PluralGoPointsFrom }}(ctx, recs)
	if err != nil {
		return
	}
	var sub{{ .PluralGoPointsFrom }} []*{{ .GoPointsFrom }}
	for _, outer := range recs {
		for i, _ := range outer.{{ .PluralGoPointsFrom }} {
			sub{{ .PluralGoPointsFrom }} = append(sub{{ .PluralGoPointsFrom }}, &outer.{{ .PluralGoPointsFrom }}[i])
		}
	}
	err = p.{{ .GoPointsFrom }}FillToDepth(ctx, sub{{ .PluralGoPointsFrom }}, maxDepth - 1)
	if err != nil {
		return
	}
{{- end }}

	return
}
{{- range .References }}

// For a give set of {{ $.GoName }}, fill in all the {{ .GoPointsFrom }}
// connected to them using a single query.
func (p *PGClient) {{ $.GoName }}Fill{{ .PluralGoPointsFrom }}(
	ctx context.Context,
	parentRecs []*{{ $.GoName }},
) error {
	ids := make([]{{ $.PkeyCol.TypeInfo.Name }}, len(parentRecs))[:0]
	idToRecord := map[{{ $.PkeyCol.TypeInfo.Name }}]*{{ $.GoName }}{}
	for i, elem := range parentRecs {
		ids = append(ids, elem.{{ $.PkeyCol.GoName }})
		idToRecord[elem.{{ $.PkeyCol.GoName }}] = parentRecs[i]
	}

	rows, err := p.DB.QueryContext(
		ctx,
		` + "`" +
	`SELECT * FROM "{{ .PgPointsFrom }}"
		 WHERE "{{ (index .PointsFromFields 0).PgName }}" = ANY($1)` +
	"`" + `,
		pq.Array(ids),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	// pull all the child records from the database and associate them with
	// the correct parent.
	for rows.Next() {
		var childRec {{ .GoPointsFrom }}
		err = childRec.Scan(rows)
		if err != nil {
			return err
		}

		parentRec := idToRecord[childRec.{{ (index .PointsFromFields 0).GoName }}]
		parentRec.{{ .PluralGoPointsFrom }} = append(parentRec.{{ .PluralGoPointsFrom }}, childRec)
	}

	return nil
}

{{ end }}
`))
