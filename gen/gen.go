package gen

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	_ "github.com/lib/pq"
)

// `pggen.Config` contains a collection of configuration options for the
// the codegenerator
type Config struct {
	// The path to a configuration file in TOML format containing information
	// about the database objects that pggen should generate code for.
	ConfigFilePath string
	// The name of the file to which the output should be written.
	OutputFileName string
	// A postgres connection string to be used to connect to the database
	ConnectionString string
	// The verbosity level of the code generator. -1 means quiet mode,
	// 0 (the default) means normal mode, and 1 means verbose mode.
	Verbosity int
}

// An instantiation of a pggen codegenerator
type Generator struct {
	// The user supplied configuration for this run of the pggen
	// codegenerator.
	config Config
	// The name of the package that all generated code is a part of.
	// Inferred from `config.OutputFileName`.
	pkg string
	// The database connection we use to gather information required
	// for code generation.
	db *sql.DB
	// The packages which need to be imported into the emitted
	// file.
	imports map[string]bool
	// The clearing house for types that we emit. They all go here
	// before being generated for real.
	types typeSet
}

// Print `output` at a normal verbosity level
func (g *Generator) info(output string) {
	if g.config.Verbosity >= 0 {
		print(output)
	}
}

// Print `output` at a normal verbosity level, formatting the output
// using the standard formatting codes from `fmt`.
func (g *Generator) infof(format string, a ...interface{}) {
	g.info(fmt.Sprintf(format, a...))
}

func FromConfig(config Config) (*Generator, error) {
	db, err := sql.Open("postgres", config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("while creating pggen Generator: %s", err.Error())
	}

	pkg, err := dirOf(config.OutputFileName)
	if err != nil {
		return nil, err
	}

	return &Generator{
		config:  config,
		db:      db,
		pkg:     pkg,
		imports: map[string]bool{},
		types:   newTypeSet(),
	}, nil
}

// Generate the code that this generator has been configured for
func (g *Generator) Gen() error {
	g.infof("using config '%s'\n", g.config.ConfigFilePath)
	confData, err := ioutil.ReadFile(g.config.ConfigFilePath)
	if err != nil {
		return err
	}

	// parse the config file
	var conf dbConfig
	tomlMd, err := toml.Decode(string(confData), &conf)
	if err != nil {
		return fmt.Errorf("while decoding config file: %s", err.Error())
	}
	for _, unknownKey := range tomlMd.Undecoded() {
		fmt.Fprintf(
			os.Stderr,
			"WARN: unknown config file key: '%s'\n",
			unknownKey.String(),
		)
	}

	// emit the prelude
	err = g.genPrelude()
	if err != nil {
		return err
	}

	//
	// Generate the code based on database objects
	//

	var body strings.Builder

	// Tables must be generated first to ensure that the type for a table is generated
	// by genTables rather than synthesized from a query result.
	err = g.genTables(&body, conf.Tables)
	if err != nil {
		return err
	}

	err = g.genStoredFuncs(&body, conf.StoredFunctions)
	if err != nil {
		return err
	}

	err = g.genQueries(&body, conf.Queries)
	if err != nil {
		return err
	}

	err = g.genStmts(&body, conf.Stmts)
	if err != nil {
		return err
	}

	//
	// Write the generate code to the file
	//

	// set up output
	outFile, err := os.OpenFile(g.config.OutputFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	out := bufio.NewWriter(outFile)
	defer out.Flush()

	// generate imports
	out.WriteString("// Code generated by pggen DO NOT EDIT\n")
	out.WriteString(fmt.Sprintf(`
package %s

import (
`, g.pkg))
	sortedPkgs := make([]string, len(g.imports))[:0]
	for pkg := range g.imports {
		sortedPkgs = append(sortedPkgs, pkg)
	}
	sort.Strings(sortedPkgs)
	for _, pkg := range sortedPkgs {
		out.WriteString(fmt.Sprintf("	%s\n", pkg))
	}
	out.WriteString(")\n\n")

	out.WriteString(body.String())

	err = g.types.gen(out)
	if err != nil {
		return err
	}

	return nil
}
