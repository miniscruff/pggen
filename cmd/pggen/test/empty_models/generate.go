package empty_models

// make sure that the schema is in place
//go:generate go run ../../../../tools/ensure-schema/main.go ../db.sql

//go:generate go run ../../main.go -o empty.gen.go pggen.toml
