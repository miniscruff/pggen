// Code generated by pggen DO NOT EDIT.

package models

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ethanpailes/pgtypes"
	"github.com/opendoor/pggen"
	"github.com/opendoor/pggen/include"
	"github.com/opendoor/pggen/unstable"
	"strings"
	"sync"
)

// PGClient wraps either a 'sql.DB' or a 'sql.Tx'. All pggen-generated
// database access methods for this package are attached to it.
type PGClient struct {
	impl       pgClientImpl
	topLevelDB pggen.DBConn

	errorConverter func(error) error

	// These column indexes are used at run time to enable us to 'SELECT *' against
	// a table that has the same columns in a different order from the ones that we
	// saw in the table we used to generate code. This means that you don't have to worry
	// about migrations merging in a slightly different order than their timestamps have
	// breaking 'SELECT *'.
	rwlockForFoo                sync.RWMutex
	colIdxTabForFoo             []int
	rwlockForGetFooValuesRow    sync.RWMutex
	colIdxTabForGetFooValuesRow []int
}

// bogus usage so we can compile with no tables configured
var _ = sync.RWMutex{}

// NewPGClient creates a new PGClient out of a '*sql.DB' or a
// custom wrapper around a db connection.
//
// If you provide your own wrapper around a '*sql.DB' for logging or
// custom tracing, you MUST forward all calls to an underlying '*sql.DB'
// member of your wrapper.
//
// If the DBConn passed into NewPGClient implements an ErrorConverter
// method which returns a func(error) error, the result of calling the
// ErrorConverter method will be called on every error that the generated
// code returns right before the error is returned. If ErrorConverter
// returns nil or is not present, it will default to the identity function.
func NewPGClient(conn pggen.DBConn) *PGClient {
	client := PGClient{
		topLevelDB: conn,
	}
	client.impl = pgClientImpl{
		db:     conn,
		client: &client,
	}

	// extract the optional error converter routine
	ec, ok := conn.(interface {
		ErrorConverter() func(error) error
	})
	if ok {
		client.errorConverter = ec.ErrorConverter()
	}
	if client.errorConverter == nil {
		client.errorConverter = func(err error) error { return err }
	}

	return &client
}

func (p *PGClient) Handle() pggen.DBHandle {
	return p.topLevelDB
}

func (p *PGClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (*TxPGClient, error) {
	tx, err := p.topLevelDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, p.errorConverter(err)
	}

	return &TxPGClient{
		impl: pgClientImpl{
			db:     tx,
			client: p,
		},
	}, nil
}

func (p *PGClient) Conn(ctx context.Context) (*ConnPGClient, error) {
	conn, err := p.topLevelDB.Conn(ctx)
	if err != nil {
		return nil, p.errorConverter(err)
	}

	return &ConnPGClient{impl: pgClientImpl{db: conn, client: p}}, nil
}

// A postgres client that operates within a transaction. Supports all the same
// generated methods that PGClient does.
type TxPGClient struct {
	impl pgClientImpl
}

func (tx *TxPGClient) Handle() pggen.DBHandle {
	return tx.impl.db.(*sql.Tx)
}

func (tx *TxPGClient) Rollback() error {
	return tx.impl.db.(*sql.Tx).Rollback()
}

func (tx *TxPGClient) Commit() error {
	return tx.impl.db.(*sql.Tx).Commit()
}

type ConnPGClient struct {
	impl pgClientImpl
}

func (conn *ConnPGClient) Close() error {
	return conn.impl.db.(*sql.Conn).Close()
}

func (conn *ConnPGClient) Handle() pggen.DBHandle {
	return conn.impl.db
}

// A database client that can wrap either a direct database connection or a transaction
type pgClientImpl struct {
	db pggen.DBHandle
	// a reference back to the owning PGClient so we can always get at the resolver tables
	client *PGClient
}

func (p *PGClient) GetFoo(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*Foo, error) {
	return p.impl.getFoo(ctx, id)
}
func (tx *TxPGClient) GetFoo(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*Foo, error) {
	return tx.impl.getFoo(ctx, id)
}
func (conn *ConnPGClient) GetFoo(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*Foo, error) {
	return conn.impl.getFoo(ctx, id)
}
func (p *pgClientImpl) getFoo(
	ctx context.Context,
	id int64,
	opts ...pggen.GetOpt,
) (*Foo, error) {
	values, err := p.listFoo(ctx, []int64{id}, true /* isGet */)
	if err != nil {
		return nil, err
	}

	// ListFoo always returns the same number of records as were
	// requested, so this is safe.
	return &values[0], err
}

func (p *PGClient) ListFoo(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []Foo, err error) {
	return p.impl.listFoo(ctx, ids, false /* isGet */, opts...)
}
func (tx *TxPGClient) ListFoo(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []Foo, err error) {
	return tx.impl.listFoo(ctx, ids, false /* isGet */, opts...)
}
func (conn *ConnPGClient) ListFoo(
	ctx context.Context,
	ids []int64,
	opts ...pggen.ListOpt,
) (ret []Foo, err error) {
	return conn.impl.listFoo(ctx, ids, false /* isGet */, opts...)
}
func (p *pgClientImpl) listFoo(
	ctx context.Context,
	ids []int64,
	isGet bool,
	opts ...pggen.ListOpt,
) (ret []Foo, err error) {
	opt := pggen.ListOptions{}
	for _, o := range opts {
		o(&opt)
	}
	if len(ids) == 0 {
		return []Foo{}, nil
	}

	rows, err := p.queryContext(
		ctx,
		`SELECT * FROM foos WHERE "id" = ANY($1)`,
		pgtypes.Array(ids),
	)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer func() {
		if err == nil {
			err = rows.Close()
			if err != nil {
				ret = nil
				err = p.client.errorConverter(err)
			}
		} else {
			rowErr := rows.Close()
			if rowErr != nil {
				err = p.client.errorConverter(fmt.Errorf("%s AND %s", err.Error(), rowErr.Error()))
			}
		}
	}()

	ret = make([]Foo, 0, len(ids))
	for rows.Next() {
		var value Foo
		err = value.Scan(ctx, p.client, rows)
		if err != nil {
			return nil, p.client.errorConverter(err)
		}
		ret = append(ret, value)
	}

	if len(ret) != len(ids) {
		if isGet {
			return nil, p.client.errorConverter(&unstable.NotFoundError{
				Msg: "GetFoo: record not found",
			})
		} else if !opt.SucceedOnPartialResults {
			return nil, p.client.errorConverter(&unstable.NotFoundError{
				Msg: fmt.Sprintf(
					"ListFoo: asked for %d records, found %d",
					len(ids),
					len(ret),
				),
			})
		}
	}

	return ret, nil
}

// Insert a Foo into the database. Returns the primary
// key of the inserted row.
func (p *PGClient) InsertFoo(
	ctx context.Context,
	value *Foo,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return p.impl.insertFoo(ctx, value, opts...)
}

// Insert a Foo into the database. Returns the primary
// key of the inserted row.
func (tx *TxPGClient) InsertFoo(
	ctx context.Context,
	value *Foo,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return tx.impl.insertFoo(ctx, value, opts...)
}

// Insert a Foo into the database. Returns the primary
// key of the inserted row.
func (conn *ConnPGClient) InsertFoo(
	ctx context.Context,
	value *Foo,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	return conn.impl.insertFoo(ctx, value, opts...)
}

// Insert a Foo into the database. Returns the primary
// key of the inserted row.
func (p *pgClientImpl) insertFoo(
	ctx context.Context,
	value *Foo,
	opts ...pggen.InsertOpt,
) (ret int64, err error) {
	var ids []int64
	ids, err = p.bulkInsertFoo(ctx, []Foo{*value}, opts...)
	if err != nil {
		return ret, p.client.errorConverter(err)
	}

	if len(ids) != 1 {
		return ret, p.client.errorConverter(fmt.Errorf("inserting a Foo: %d ids (expected 1)", len(ids)))
	}

	ret = ids[0]
	return
}

// Insert a list of Foo. Returns a list of the primary keys of
// the inserted rows.
func (p *PGClient) BulkInsertFoo(
	ctx context.Context,
	values []Foo,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return p.impl.bulkInsertFoo(ctx, values, opts...)
}

// Insert a list of Foo. Returns a list of the primary keys of
// the inserted rows.
func (tx *TxPGClient) BulkInsertFoo(
	ctx context.Context,
	values []Foo,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return tx.impl.bulkInsertFoo(ctx, values, opts...)
}

// Insert a list of Foo. Returns a list of the primary keys of
// the inserted rows.
func (conn *ConnPGClient) BulkInsertFoo(
	ctx context.Context,
	values []Foo,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	return conn.impl.bulkInsertFoo(ctx, values, opts...)
}

// Insert a list of Foo. Returns a list of the primary keys of
// the inserted rows.
func (p *pgClientImpl) bulkInsertFoo(
	ctx context.Context,
	values []Foo,
	opts ...pggen.InsertOpt,
) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}

	opt := pggen.InsertOptions{}
	for _, o := range opts {
		o(&opt)
	}

	defaultFields := opt.DefaultFields.Intersection(defaultableColsForFoo)
	args := make([]interface{}, 0, 2*len(values))
	for _, v := range values {
		if opt.UsePkey && !defaultFields.Test(FooIdFieldIndex) {
			args = append(args, v.Id)
		}
		if !defaultFields.Test(FooValueFieldIndex) {
			args = append(args, v.Value)
		}
	}

	bulkInsertQuery := genBulkInsertStmt(
		`foos`,
		fieldsForFoo,
		len(values),
		"id",
		opt.UsePkey,
		defaultFields,
	)

	rows, err := p.queryContext(ctx, bulkInsertQuery, args...)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer rows.Close()

	ids := make([]int64, 0, len(values))
	for rows.Next() {
		var id int64
		err = rows.Scan(&(id))
		if err != nil {
			return nil, p.client.errorConverter(err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// bit indicies for 'fieldMask' parameters
const (
	FooIdFieldIndex    int = 0
	FooValueFieldIndex int = 1
	FooMaxFieldIndex   int = (2 - 1)
)

// A field set saying that all fields in Foo should be updated.
// For use as a 'fieldMask' parameter
var FooAllFields pggen.FieldSet = pggen.NewFieldSetFilled(2)

var defaultableColsForFoo = func() pggen.FieldSet {
	fs := pggen.NewFieldSet(FooMaxFieldIndex)
	fs.Set(FooIdFieldIndex, true)
	return fs
}()

var fieldsForFoo []fieldNameAndIdx = []fieldNameAndIdx{
	{name: `id`, idx: FooIdFieldIndex},
	{name: `value`, idx: FooValueFieldIndex},
}

// Update a Foo. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (p *PGClient) UpdateFoo(
	ctx context.Context,
	value *Foo,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return p.impl.updateFoo(ctx, value, fieldMask, opts...)
}

// Update a Foo. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (tx *TxPGClient) UpdateFoo(
	ctx context.Context,
	value *Foo,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return tx.impl.updateFoo(ctx, value, fieldMask, opts...)
}

// Update a Foo. 'value' must at the least have
// a primary key set. The 'fieldMask' field set indicates which fields
// should be updated in the database.
//
// Returns the primary key of the updated row.
func (conn *ConnPGClient) UpdateFoo(
	ctx context.Context,
	value *Foo,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	return conn.impl.updateFoo(ctx, value, fieldMask, opts...)
}
func (p *pgClientImpl) updateFoo(
	ctx context.Context,
	value *Foo,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpdateOpt,
) (ret int64, err error) {
	opt := pggen.UpdateOptions{}
	for _, o := range opts {
		o(&opt)
	}

	if !fieldMask.Test(FooIdFieldIndex) {
		return ret, p.client.errorConverter(fmt.Errorf(`primary key required for updates to 'foos'`))
	}

	updateStmt := genUpdateStmt(
		`foos`,
		"id",
		fieldsForFoo,
		fieldMask,
		"id",
	)

	args := make([]interface{}, 0, 2)
	if fieldMask.Test(FooIdFieldIndex) {
		args = append(args, value.Id)
	}
	if fieldMask.Test(FooValueFieldIndex) {
		args = append(args, value.Value)
	}

	// add the primary key arg for the WHERE condition
	args = append(args, value.Id)

	var id int64
	err = p.db.QueryRowContext(ctx, updateStmt, args...).
		Scan(&(id))
	if err != nil {
		return ret, p.client.errorConverter(err)
	}

	return id, nil
}

// Upsert a Foo value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (p *PGClient) UpsertFoo(
	ctx context.Context,
	value *Foo,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = p.impl.bulkUpsertFoo(ctx, []Foo{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a Foo value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (tx *TxPGClient) UpsertFoo(
	ctx context.Context,
	value *Foo,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = tx.impl.bulkUpsertFoo(ctx, []Foo{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a Foo value. If the given value conflicts with
// an existing row in the database, use the provided value to update that row
// rather than inserting it. Only the fields specified by 'fieldMask' are
// actually updated. All other fields are left as-is.
func (conn *ConnPGClient) UpsertFoo(
	ctx context.Context,
	value *Foo,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret int64, err error) {
	var val []int64
	val, err = conn.impl.bulkUpsertFoo(ctx, []Foo{*value}, constraintNames, fieldMask, opts...)
	if err != nil {
		return
	}
	if len(val) == 1 {
		return val[0], nil
	}

	// only possible if no upsert fields were specified by the field mask
	return value.Id, nil
}

// Upsert a set of Foo values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (p *PGClient) BulkUpsertFoo(
	ctx context.Context,
	values []Foo,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return p.impl.bulkUpsertFoo(ctx, values, constraintNames, fieldMask, opts...)
}

// Upsert a set of Foo values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (tx *TxPGClient) BulkUpsertFoo(
	ctx context.Context,
	values []Foo,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return tx.impl.bulkUpsertFoo(ctx, values, constraintNames, fieldMask, opts...)
}

// Upsert a set of Foo values. If any of the given values conflict with
// existing rows in the database, use the provided values to update the rows which
// exist in the database rather than inserting them. Only the fields specified by
// 'fieldMask' are actually updated. All other fields are left as-is.
func (conn *ConnPGClient) BulkUpsertFoo(
	ctx context.Context,
	values []Foo,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) (ret []int64, err error) {
	return conn.impl.bulkUpsertFoo(ctx, values, constraintNames, fieldMask, opts...)
}
func (p *pgClientImpl) bulkUpsertFoo(
	ctx context.Context,
	values []Foo,
	constraintNames []string,
	fieldMask pggen.FieldSet,
	opts ...pggen.UpsertOpt,
) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}

	options := pggen.UpsertOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if constraintNames == nil || len(constraintNames) == 0 {
		constraintNames = []string{`id`}
	}

	defaultFields := options.DefaultFields.Intersection(defaultableColsForFoo)
	var stmt strings.Builder
	genInsertCommon(
		&stmt,
		`foos`,
		fieldsForFoo,
		len(values),
		`id`,
		options.UsePkey,
		defaultFields,
	)

	setBits := fieldMask.CountSetBits()
	hasConflictAction := setBits > 1 ||
		(setBits == 1 && fieldMask.Test(FooIdFieldIndex) && options.UsePkey) ||
		(setBits == 1 && !fieldMask.Test(FooIdFieldIndex))

	if hasConflictAction {
		stmt.WriteString("ON CONFLICT (")
		stmt.WriteString(strings.Join(constraintNames, ","))
		stmt.WriteString(") DO UPDATE SET ")

		updateCols := make([]string, 0, 2)
		updateExprs := make([]string, 0, 2)
		if options.UsePkey {
			updateCols = append(updateCols, `id`)
			updateExprs = append(updateExprs, `excluded.id`)
		}
		if fieldMask.Test(FooValueFieldIndex) {
			updateCols = append(updateCols, `value`)
			updateExprs = append(updateExprs, `excluded.value`)
		}
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateCols, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
		stmt.WriteString(" = ")
		if len(updateCols) > 1 {
			stmt.WriteRune('(')
		}
		stmt.WriteString(strings.Join(updateExprs, ","))
		if len(updateCols) > 1 {
			stmt.WriteRune(')')
		}
	} else {
		stmt.WriteString("ON CONFLICT DO NOTHING")
	}

	stmt.WriteString(` RETURNING "id"`)

	args := make([]interface{}, 0, 2*len(values))
	for _, v := range values {
		if options.UsePkey && !defaultFields.Test(FooIdFieldIndex) {
			args = append(args, v.Id)
		}
		if !defaultFields.Test(FooValueFieldIndex) {
			args = append(args, v.Value)
		}
	}

	rows, err := p.queryContext(ctx, stmt.String(), args...)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer rows.Close()

	ids := make([]int64, 0, len(values))
	for rows.Next() {
		var id int64
		err = rows.Scan(&(id))
		if err != nil {
			return nil, p.client.errorConverter(err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (p *PGClient) DeleteFoo(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteFoo(ctx, []int64{id}, opts...)
}
func (tx *TxPGClient) DeleteFoo(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteFoo(ctx, []int64{id}, opts...)
}
func (conn *ConnPGClient) DeleteFoo(
	ctx context.Context,
	id int64,
	opts ...pggen.DeleteOpt,
) error {
	return conn.impl.bulkDeleteFoo(ctx, []int64{id}, opts...)
}

func (p *PGClient) BulkDeleteFoo(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return p.impl.bulkDeleteFoo(ctx, ids, opts...)
}
func (tx *TxPGClient) BulkDeleteFoo(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return tx.impl.bulkDeleteFoo(ctx, ids, opts...)
}
func (conn *ConnPGClient) BulkDeleteFoo(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	return conn.impl.bulkDeleteFoo(ctx, ids, opts...)
}
func (p *pgClientImpl) bulkDeleteFoo(
	ctx context.Context,
	ids []int64,
	opts ...pggen.DeleteOpt,
) error {
	if len(ids) == 0 {
		return nil
	}

	options := pggen.DeleteOptions{}
	for _, o := range opts {
		o(&options)
	}
	res, err := p.db.ExecContext(
		ctx,
		`DELETE FROM foos WHERE "id" = ANY($1)`,
		pgtypes.Array(ids),
	)
	if err != nil {
		return p.client.errorConverter(err)
	}

	nrows, err := res.RowsAffected()
	if err != nil {
		return p.client.errorConverter(err)
	}

	if nrows != int64(len(ids)) {
		return p.client.errorConverter(fmt.Errorf(
			"BulkDeleteFoo: %d rows deleted, expected %d",
			nrows,
			len(ids),
		))
	}

	return err
}

var FooAllIncludes *include.Spec = include.Must(include.Parse(
	`foos`,
))

func (p *PGClient) FooFillIncludes(
	ctx context.Context,
	rec *Foo,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateFooBulkFillIncludes(ctx, []*Foo{rec}, includes)
}
func (tx *TxPGClient) FooFillIncludes(
	ctx context.Context,
	rec *Foo,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateFooBulkFillIncludes(ctx, []*Foo{rec}, includes)
}
func (conn *ConnPGClient) FooFillIncludes(
	ctx context.Context,
	rec *Foo,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return conn.impl.privateFooBulkFillIncludes(ctx, []*Foo{rec}, includes)
}

func (p *PGClient) FooBulkFillIncludes(
	ctx context.Context,
	recs []*Foo,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return p.impl.privateFooBulkFillIncludes(ctx, recs, includes)
}
func (tx *TxPGClient) FooBulkFillIncludes(
	ctx context.Context,
	recs []*Foo,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return tx.impl.privateFooBulkFillIncludes(ctx, recs, includes)
}
func (conn *ConnPGClient) FooBulkFillIncludes(
	ctx context.Context,
	recs []*Foo,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	return conn.impl.privateFooBulkFillIncludes(ctx, recs, includes)
}
func (p *pgClientImpl) privateFooBulkFillIncludes(
	ctx context.Context,
	recs []*Foo,
	includes *include.Spec,
	opts ...pggen.IncludeOpt,
) error {
	loadedRecordTab := map[string]interface{}{}

	return p.implFooBulkFillIncludes(ctx, recs, includes, loadedRecordTab)
}

func (p *pgClientImpl) implFooBulkFillIncludes(
	ctx context.Context,
	recs []*Foo,
	includes *include.Spec,
	loadedRecordTab map[string]interface{},
) (err error) {
	if includes.TableName != `foos` {
		return p.client.errorConverter(fmt.Errorf(
			`expected includes for 'foos', got '%s'`,
			includes.TableName,
		))
	}

	loadedTab, inMap := loadedRecordTab[`foos`]
	if inMap {
		idToRecord := loadedTab.(map[int64]*Foo)
		for _, r := range recs {
			_, alreadyLoaded := idToRecord[r.Id]
			if !alreadyLoaded {
				idToRecord[r.Id] = r
			}
		}
	} else {
		idToRecord := make(map[int64]*Foo, len(recs))
		for _, r := range recs {
			idToRecord[r.Id] = r
		}
		loadedRecordTab[`foos`] = idToRecord
	}

	return
}

func (p *PGClient) GetFooValues(
	ctx context.Context,
	arg1 []int64,
) (ret []*string, err error) {
	return p.impl.GetFooValues(
		ctx,
		arg1,
	)
}

func (tx *TxPGClient) GetFooValues(
	ctx context.Context,
	arg1 []int64,
) (ret []*string, err error) {
	return tx.impl.GetFooValues(
		ctx,
		arg1,
	)
}

func (conn *ConnPGClient) GetFooValues(
	ctx context.Context,
	arg1 []int64,
) (ret []*string, err error) {
	return conn.impl.GetFooValues(
		ctx,
		arg1,
	)
}
func (p *pgClientImpl) GetFooValues(
	ctx context.Context,
	arg1 []int64,
) (ret []*string, err error) {
	ret = []*string{}

	var rows *sql.Rows
	rows, err = p.GetFooValuesQuery(
		ctx,
		arg1,
	)
	if err != nil {
		return nil, p.client.errorConverter(err)
	}
	defer func() {
		if err == nil {
			err = rows.Close()
			if err != nil {
				ret = nil
				err = p.client.errorConverter(err)
			}
		} else {
			rowErr := rows.Close()
			if rowErr != nil {
				err = p.client.errorConverter(fmt.Errorf("%s AND %s", err.Error(), rowErr.Error()))
			}
		}
	}()

	for rows.Next() {
		var row *string
		var scanTgt sql.NullString
		err = rows.Scan(&(scanTgt))
		if err != nil {
			return nil, p.client.errorConverter(err)
		}
		row = convertNullString(scanTgt)
		ret = append(ret, row)
	}

	return
}

func (p *PGClient) GetFooValuesQuery(
	ctx context.Context,
	arg1 []int64,
) (*sql.Rows, error) {
	return p.impl.GetFooValuesQuery(
		ctx,
		arg1,
	)
}

func (tx *TxPGClient) GetFooValuesQuery(
	ctx context.Context,
	arg1 []int64,
) (*sql.Rows, error) {
	return tx.impl.GetFooValuesQuery(
		ctx,
		arg1,
	)
}

func (conn *ConnPGClient) GetFooValuesQuery(
	ctx context.Context,
	arg1 []int64,
) (*sql.Rows, error) {
	return conn.impl.GetFooValuesQuery(
		ctx,
		arg1,
	)
}
func (p *pgClientImpl) GetFooValuesQuery(
	ctx context.Context,
	arg1 []int64,
) (*sql.Rows, error) {
	return p.queryContext(
		ctx,
		`SELECT value FROM foos WHERE id = ANY($1)`,
		pgtypes.Array(arg1),
	)
}

type DBQueries interface {
	//
	// automatic CRUD methods
	//

	// Foo methods
	GetFoo(ctx context.Context, id int64, opts ...pggen.GetOpt) (*Foo, error)
	ListFoo(ctx context.Context, ids []int64, opts ...pggen.ListOpt) ([]Foo, error)
	InsertFoo(ctx context.Context, value *Foo, opts ...pggen.InsertOpt) (int64, error)
	BulkInsertFoo(ctx context.Context, values []Foo, opts ...pggen.InsertOpt) ([]int64, error)
	UpdateFoo(ctx context.Context, value *Foo, fieldMask pggen.FieldSet, opts ...pggen.UpdateOpt) (ret int64, err error)
	UpsertFoo(ctx context.Context, value *Foo, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) (int64, error)
	BulkUpsertFoo(ctx context.Context, values []Foo, constraintNames []string, fieldMask pggen.FieldSet, opts ...pggen.UpsertOpt) ([]int64, error)
	DeleteFoo(ctx context.Context, id int64, opts ...pggen.DeleteOpt) error
	BulkDeleteFoo(ctx context.Context, ids []int64, opts ...pggen.DeleteOpt) error
	FooFillIncludes(ctx context.Context, rec *Foo, includes *include.Spec, opts ...pggen.IncludeOpt) error
	FooBulkFillIncludes(ctx context.Context, recs []*Foo, includes *include.Spec, opts ...pggen.IncludeOpt) error

	//
	// query methods
	//

	// GetFooValues query
	GetFooValues(
		ctx context.Context,
		arg1 []int64,
	) ([]*string, error)
	GetFooValuesQuery(
		ctx context.Context,
		arg1 []int64,
	) (*sql.Rows, error)

	//
	// stored function methods
	//

	//
	// stmt methods
	//

}

type Foo struct {
	Id    int64   `gorm:"column:id;is_primary"`
	Value *string `gorm:"column:value"`
}

func (r *Foo) Scan(ctx context.Context, client *PGClient, rs *sql.Rows) error {
	client.rwlockForFoo.RLock()
	if client.colIdxTabForFoo == nil {
		client.rwlockForFoo.RUnlock() // release the lock to allow the write lock to be aquired
		err := client.fillColPosTab(
			ctx,
			genTimeColIdxTabForFoo,
			&client.rwlockForFoo,
			rs,
			&client.colIdxTabForFoo,
		)
		if err != nil {
			return err
		}
		client.rwlockForFoo.RLock() // get the lock back for the rest of the routine
	}

	var nullableTgts nullableScanTgtsForFoo

	scanTgts := make([]interface{}, len(client.colIdxTabForFoo))
	for runIdx, genIdx := range client.colIdxTabForFoo {
		if genIdx == -1 {
			scanTgts[runIdx] = &pggenSinkScanner{}
		} else {
			scanTgts[runIdx] = scannerTabForFoo[genIdx](r, &nullableTgts)
		}
	}
	client.rwlockForFoo.RUnlock() // we are now done referencing the idx tab in the happy path

	err := rs.Scan(scanTgts...)
	if err != nil {
		// The database schema may have been changed out from under us, let's
		// check to see if we just need to update our column index tables and retry.
		colNames, colsErr := rs.Columns()
		if colsErr != nil {
			return fmt.Errorf("pggen: checking column names: %s", colsErr.Error())
		}
		client.rwlockForFoo.RLock()
		if len(client.colIdxTabForFoo) != len(colNames) {
			client.rwlockForFoo.RUnlock() // release the lock to allow the write lock to be aquired
			err = client.fillColPosTab(
				ctx,
				genTimeColIdxTabForFoo,
				&client.rwlockForFoo,
				rs,
				&client.colIdxTabForFoo,
			)
			if err != nil {
				return err
			}

			return r.Scan(ctx, client, rs)
		} else {
			client.rwlockForFoo.RUnlock()
			return err
		}
	}
	r.Value = convertNullString(nullableTgts.scanValue)

	return nil
}

type nullableScanTgtsForFoo struct {
	scanValue sql.NullString
}

// a table mapping codegen-time col indicies to functions returning a scanner for the
// field that was at that column index at codegen-time.
var scannerTabForFoo = [...]func(*Foo, *nullableScanTgtsForFoo) interface{}{
	func(
		r *Foo,
		nullableTgts *nullableScanTgtsForFoo,
	) interface{} {
		return &(r.Id)
	},
	func(
		r *Foo,
		nullableTgts *nullableScanTgtsForFoo,
	) interface{} {
		return &(nullableTgts.scanValue)
	},
}

var genTimeColIdxTabForFoo map[string]int = map[string]int{
	`id`:    0,
	`value`: 1,
}
var _ = unstable.NotFoundError{}
