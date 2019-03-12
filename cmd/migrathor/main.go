// Migrathor is a cli tool for schema migration handling in PostgreSQL.
//
// Complete documentation is available at https://github.com/denisbrodbeck/migrathor/.
//
// Usage:
//
// 	migrathor <command> [arguments]
//
// The commands are
//
// 	create     create a new migration file
// 	migrate    run the database migrations
// 	version    print migrathor version
//
// The arguments are
//
// 	-path         path to the migrations files to be executed (default migrations)
// 	-table        name of applied migrations history table (default migrations)
// 	-host         database hostname (default localhost)
// 	-port         database port (default 5432)
// 	-name         database name (default postgres)
// 	-user         database user (default postgres)
// 	-pass         database password (default empty)
// 	-timeout      connection timeout in seconds (default 10s)
// 	-sslmode      SSL mode (default disable - see [SSL modes])
// 	-sslcert      PEM encoded cert file location
// 	-sslkey       PEM encoded key file location
// 	-sslrootcert  PEM encoded root certificate file location
//
// Available SSL modes
//
// 	disable      no SSL
// 	require      always SSL (skip verification)
// 	verify-ca    always SSL (verify server cert was signed by a trusted CA)
// 	verify-full  always SSL (verify server cert matches hostname and was signed by a trusted CA)
package main

import (
	"os"
)

var usage = `
migrathor is a schema migration handler for PostgreSQL.

Complete documentation is available at https://github.com/denisbrodbeck/migrathor/.

Usage:

	migrathor <command> [arguments]

The commands are:

	create     create a new migration file
	migrate    run the database migrations
	version    print migrathor version

The arguments are:

	-path         path to the migrations files to be executed (default migrations)
	-table        name of applied migrations history table (default migrations)
	-host         database hostname (default localhost)
	-port         database port (default 5432)
	-name         database name (default postgres)
	-user         database user (default postgres)
	-pass         database password (default empty)
	-timeout      connection timeout in seconds (default 10s)
	-sslmode      SSL mode (default disable - see [SSL modes])
	-sslcert      PEM encoded cert file location
	-sslkey       PEM encoded key file location
	-sslrootcert  PEM encoded root certificate file location

Available SSL modes:

	disable      no SSL
	require      always SSL (skip verification)
	verify-ca    always SSL (verify server cert was signed by a trusted CA)
	verify-full  always SSL (verify server cert matches hostname and was signed by a trusted CA)
`[1:]

// set by ldflags when built
var (
	gitTag = "<not set>"
)

func main() {
	// main() is untestable --> do any work outside of main()
	os.Exit(ParseAndRun(os.Stdout, os.Stderr, os.Stdin, os.Args[1:]))
}
