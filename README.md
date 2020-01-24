# pggen

`lib/pggen` is the back end for the `tools/pggen` command line tool which
can be used for database interactivity. All command line options have
corresponding fields in `pggen.Config`, so you can run the codegenerator
programatically if you prefer.

# Development

The tests for `lib/pggen` live in the `tools/pggen` tree. See the
README in that directory for more information on developing pggen.

## Testing

To test pggen, you will need a `pggen_development` postgres database. If you
don't already have one, run `createdb pggen_development`. To populate the
test database, run `psql pggen_development < test/db.sql`. Once you have
the database set up, you can run tests with `go generate ./... && go test ./...`.

# Using `pggen`

## Features

`pggen` offers two main features: automatic generation of shims wrapping
SQL queries and automatic generation of go structs from SQL tables.

### Query Shims

`pggen` knows how to infer the return and argument types of queries, so all
you have to write is the SQL that you want to execute using standard postgres
$N placeholder syntax for parameters and pggen will generate simple go
wrapper functions that perform all the boilerplate needed to call the
query from go. `pggen` will automatically generate a struct to contain
the result rows that the query returns, though if you want to re-use
a return type between queries you can do so by providing a return
type name.

If you have the following entry in your toml file

```toml
[[query]]
name = "GetIdAndCreated"
body = '''
SELECT id, created_at
FROM flip
WHERE ID = $1
ORDER BY created_at
'''
```

and the following DDL to define your database schema

```sql
CREATE TABLE flip (
	id SERIAL PRIMARY KEY NOT NULL,
    created_at TIMESTAMP,
    ...
);
```

`pggen` will generate a return type

```go
type GetIdAndCreatedRow struct {
	Id *int64 `gorm:"column:id" gorm:"is_primary"`
	CreatedAt *time.Time `gorm:"column:created_at"`
}
```

for you. `GetIdAndCreated` will have a `Scan` method which accepts a
`*sql.Rows` as and argument. `pggen` will also generate two functions
`GetIdAndCreated` and `GetIdAndCreatedQuery`.

`GetIDAndCreated` should be your go-to method for invoking this query.
It accepts a `context.Context` and all the arguments to the query, in this case
just a single `int64` argument, and returns a slice of `GetIdAndCreatedRow`s along
with a possible error. This essentially turns an SQL query into a type safe RPC call
to the database.

Sometimes you don't want to load all of the results of a query into memory at once,
in which case `GetIdAndCreatedQuery` is useful. It accepts the same arguments as
`GetIdAndCreated` and returns `(*sql.Rows, error)`. This isn't much higher level than
just placing the SQL call directly, but you still retain the benefit of having type
safe query parameters. Once you have the `*sql.Rows` in hand, you can make use of
the `Scan` method on `GetIdAndCreatedRow` to lazily load query results in a loop.

#### Named Return Types

If you don't provide a name for your return type `pggen` is happy to
make one up, but if you want to override the fairly uninspired name
that `pggen` will come up with, you can do so. This feature is also
the key to processing the result types of multiple queries with the
same code. In the above example, this would allow you to override
the name of `GetIdAndCreatedRow`.

#### Null Flags

Postgres does not perform inference about the nullability of the fields
returned via a query, so by default `pggen` will generate boxed fields
for the return struct. If you know for sure that certain query result
fields cannot ever be null, you may use the `null_flags`
configuration option to tell `pggen` not to box the fields in question.
Be sure to apply this flag consistently when dealing with columns that appear
in multiple `pggen` queries, as both the field names and their nullability
values must match up in order for the generated type to be reused.

The value of the null flags configuration option, if it is provided, should
be a string that is exactly as long as the number of fields that the query
returns. For each field in the return type, the character at the corresponding
position in the null flags string indicates the nullability of the field, with
'-' meaning that the field is NOT NULL and 'n' indicating that the field is
nullable.

If you knew for a fact that the `id` and `created_at` fields could not be null
in the above example, you could modify your toml entry to read

```toml
[[query]]
name = "GetIdAndCreated"
body = '''
SELECT id, created_at
FROM flip
WHERE ID = $1
ORDER BY created_at
'''
null_flags = "--"
```

or equivalently

```toml
[[query]]
name = "GetIdAndCreated"
body = '''
SELECT id, created_at
FROM flip
WHERE ID = $1
ORDER BY created_at
'''
not_null_fields = ["id", "created_at"]
```

which would cause the result type

```go
type GetIdAndCreatedRow struct {
	Id int64 `gorm:"column:id" gorm:"is_primary"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
```

to be generated. Note the fact that the fields are no longer boxed.

### Model Structs

`pggen` translates table definitions into golang structs along with
a stable of common CRUD operations for working with those structs.
In addition to the provided CRUD operations, you can use the model
structs generated by `pggen` as return values from your own custom
queries. You can also easily use your own custom dynamically
generated SQL to produce model structs by making use of the `Scan` method
attached to all of them.

#### Generated Struct

The generated struct for a postgres table is very similar to the generated
struct for a query return value. Postgres does expose the nullability of table
columns, so you don't have to worry about setting explicitly setting null flags
for a table.

If you had the DDL

```sql
CREATE TABLE small_entities (
	id SERIAL PRIMARY KEY NOT NULL,
    anint integer NOT NULL
);

CREATE TABLE attachments (
    id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    small_entity_id integer NOT NULL
        REFERENCES small_entities(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    value text
);
```

and the following entries in your toml file

```toml
[[table]]
name = "small_entities"

[[table]]
name = "attachments"
```

`pggen` would generate the following structs for the two different tables.

```golang
type SmallEntity struct {
	Id int64 `gorm:"column:id" gorm:"is_primary"`
	Anint int64 `gorm:"column:anint"`
	Attachments []Attachment
}

type Attachment struct {
	Id uuid.UUID `gorm:"column:id" gorm:"is_primary"`
	SmallEntityId int64 `gorm:"column:small_entity_id"`
	Value *string `gorm:"column:value"`
}
```

There are a few things worth noting here. First, the structs are not named exactly the
same thing as the tables which they are generated from. Tables conventionally have plural
names, while golang structs conventionally have singular names, so `pggen` will convert
plural table names to singular names. This is important for interop with `gorm`, as `gorm`
imposes the same rule on table vs struct names. In fact, `pggen` and `gorm` use exactly
the [same code](https://github.com/jinzhu/inflection) to determine which names are plural
and which are singular.

The second thing to note here is that `SmallEntity` has an `Attachments` field, which doesn't
show up in the DDL for the database tables. This is because `pggen` has noticed the foreign
key constraint on the `attachments` table and inferred that `Attachment` is a child entity
of `SmallEntity`. Child entities are not automatically filled in by the various accessors
methods when a record is fetched from the database, but `pggen` does provide utility code
which can be invoked for populating them. Child entities are only attached to a generated
struct if the table which holds the foreign key is also registered in the toml file.

Lastly, struct fields are generated as either boxed or unboxed types depending on the
nullability of the corresponding columns in the DDL.

#### Generated Methods & Values

Below is a list of the methods on `PGClient` which are generated for each table registered
in the configuration file

- Methods
    - Get<Entity>
        - Given the primary key of an entity, Get<Entity> fetches the entity with that key.
    - List<Entity>
        - Given a list of primary keys, List<Entity> returns a unordered list of entities
          with the given primary keys. List<Entity> always returns either exactly as many
          entities as were requested or an error (i.e. partial successes are treated as failures).
    - Insert<Entity>
        - Given an entity struct, Insert<Entity> inserts it into the database and returns
          the primary key of the inserted struct, or an error if the insert operation failed.
    - BulkInsert<Entity>
        - Given a list of entity structs, BulkInsert<Entity> inserts them all and returns
          the primary keys of the inserted structs. Note that it is possible for only a subset
          of the rows to be inserted if inserting some rows would violate existing database constraints.
          If the insert needs to be fully atomic, you can wrap the call to BulkInsert in a transaction.
    - Update<Entity>
        - Given an entity struct and a bitset, Update<Entity> updates all the fields of the
          given struct with their corresponding bit set in the database and returns the
          primary key of the updated record.
    - Delete<Entity>
        - Given the id of an entity, Delete<Entity> deletes it and returns an error on failure or
          nil on success.
    - <Entity>FillAll
        - Given an entity, <Entity>FillAll fills in all the attached decendant entities.
          For entities without children, this is a no-op. This method is recursive, so
          grandchildren and great-grandchildren will be filled in as well.
          It returns an error on failure and nil on success.
    - <Entity>FillToDepth
        - Given a list of pointers to entities and a max depth, <Entity>FillToDepth fills
          in all the decendant entities recursivly up to the provided max depth. This is mostly intended
          as an internal routine, but it may be useful for clients to have more granular
          control over how child entities are filled in so it is publicly exposed.
          For entities without children, this is a no-op. It returns an error on failure and
          nil on success.
    - <Entity>Fill<ChildEntities>
        - Given a list of pointers to entities, <Entity>Fill<ChildEntities> fills in all of
          the child entities of a specific type attached to the entities in the list. Just like
          <Entity>FillToDepth, this is mostly an internal routine that is exposed for cases
          when more control is needed. It returns an error on failure and nil on success.
- Values (constant or variable definitions)
    - <Entity><FieldName>FieldIndex
        - For each field in the entity, `pggen` generates a constant indicating the field's
          index (0-based). These constants are useful when working with the bitset that gets
          passed to Update<Entity>.
    - <Entity>AllFields
        - A bitset with the bits for all the fields in <Entity> set

### [Prepared Functions](https://www.postgresql.org/docs/current/sql-createfunction.html)

Postgres provides a feature called prepared functions, allowing you to register a function
in the database ahead of time and then call it from queries or other functions. `pggen` provides
support for generating shims for prepared functions. Prepared functions don't provide all
that much functionality over and above that provided by `pggen`s support for queries. The
main advantage is that the argument names for the shims will be pulled from the argument names
of the prepared function in postgres rather than being a fairly opaque `arg0`, `arg1`, ...,
`argn`.

### Statements

Sometimes you want to execute SQL commands for side effects rather than for a set of
query results. To support these use cases `pggen` supports registering statements in
the config file. Shims generated for statements return `(sql.Result, error)` rather than
a slice of query results or an error. For example, to perform a custom insert you might write
the following in your config file

```toml
[[statement]]
name = "MyInsertSmallEntity"
body = '''
INSERT INTO small_entities (anint) VALUES ($1)
'''
```

which would generate a shim with the signature

```
MyInsertSmallEntity(ctx context.Context, arg0 int64) (sql.Result, error)
```

### GORM Compatibility

`pggen` aims to generate models which are compatible with the `gorm` tool. We have a lot of
code which uses `gorm` already and some people may prefer using `gorm` over the routines that
`pggen` provides. `pggen` can still help those people by taking care of the drudge work of
writing model structs which match up with the database table definitions.

## Configuration

`pggen` is configured with a `toml` file. Some of the configuration options have already
been mentioned in this document, but the most complete source of documentation
on is the comments in `$CODE/go/lib/pggen/gen/config_file.go`.
An example file can be found at `$CODE/go/tools/pggen/test/pggen.toml`.
