package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/denisbrodbeck/migrathor"
	"github.com/peterbourgon/ff"

	_ "github.com/lib/pq"
)

// ParseAndRun parses the command line, and then runs the passed commands.
func ParseAndRun(stdout, stderr io.Writer, stdin io.Reader, args []string) int {
	out := log.New(stdout, "", 0)
	errlog := log.New(stderr, "", 0)

	fs := flag.NewFlagSet("migrathor", flag.ContinueOnError)
	fs.Usage = func() {
		fs.Output().Write([]byte(usage))
	}
	var (
		flagPath        = fs.String("path", "migrations", "the path to the migrations files to be executed")
		flagTable       = fs.String("table", "migrations", "name of applied migrations history table")
		flagHost        = fs.String("host", "localhost", "database host")
		flagPort        = fs.String("port", "5432", "database port")
		flagName        = fs.String("name", "postgres", "database name")
		flagUser        = fs.String("user", "postgres", "database user")
		flagPass        = fs.String("pass", "", "database password")
		flagTimeout     = fs.Duration("timeout", time.Second*10, "connection timeout in seconds (default 10s)")
		flagSSLMode     = fs.String("sslmode", "disable", "database SSL mode (see options)")
		flagSSLCert     = fs.String("sslcert", "", "PEM encoded cert file location")
		flagSSLKey      = fs.String("sslkey", "", "PEM encoded key file location")
		flagSSLRootCert = fs.String("sslrootcert", "", "PEM encoded root certificate file location")
	)
	err := ff.Parse(fs, args, ff.WithEnvVarPrefix("MIGRATHOR"))
	if err != nil {
		if err != flag.ErrHelp {
			fs.Output().Write([]byte(fmt.Sprintf("\nUsage error: %s\n", err)))
		}
		return 1
	}

	// wire up miration with user-provided migration table und connect library logger to stdout
	table := func(m *migrathor.Migration) {
		m.Table = *flagTable
	}
	logger := func(m *migrathor.Migration) {
		m.Logger = out.Print
	}
	migration := migrathor.New(*flagPath, table, logger)

	// parse commands
	commands := fs.Args()
	if len(commands) >= 1 {
		switch strings.ToLower(commands[0]) {
		case "create":
			name := "placeholder"
			if len(commands) >= 2 {
				name = commands[1]
			}

			path, err := migration.Create(name)
			if err != nil {
				errlog.Println(err)
				return 3
			}
			log.Printf("Created Migration: %s", path)
		case "migrate":
			db, err := connect(createDSN(*flagHost, *flagPort, *flagName, *flagUser, *flagPass, *flagSSLMode, *flagSSLCert, *flagSSLKey, *flagSSLRootCert, *flagTimeout))
			if err != nil {
				errlog.Println(err)
				return 2
			}
			defer logCloser(db, errlog)

			// give a generous timeout of 5 minutes
			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute*5)
			defer cancelFunc()

			applied, err := migration.Apply(ctx, db)
			if err != nil {
				errlog.Printf("failed to run migrations: %v", err)
				if pqerr := migrathor.UnderlyingError(err); pqerr != err {
					errlog.Println(formatPqError(pqerr))
				}
			}
			if applied != nil && len(applied) > 0 {
				for _, mig := range applied {
					out.Printf("applied: %s\n", mig)
				}
				out.Printf("Applied migrations: %d\n", len(applied))
			}
		case "version":
			out.Println(gitTag)
		}
	}

	return 0
}

func createDSN(host, port, name, user, pass, sslmode, sslcert, sslkey, sslrootcert string, timeout time.Duration) string {
	dsn := ""
	if host != "" {
		dsn += fmt.Sprintf("host=%s ", host)
	}
	if port != "" {
		dsn += fmt.Sprintf("port=%s ", port)
	}
	if name != "" {
		dsn += fmt.Sprintf("dbname='%s' ", name)
	}
	if user != "" {
		dsn += fmt.Sprintf("user='%s' ", user)
	}
	if pass != "" {
		// values with spaces must be surrounded with '': e.g. 'se cret'
		// further ' within the value must be escaped with \
		password := strings.Replace(pass, "'", `\'`, -1)
		dsn += fmt.Sprintf("password='%s' ", password)
	}
	if sslmode != "" {
		dsn += fmt.Sprintf("sslmode=%s ", sslmode)
	}
	if sslcert != "" {
		dsn += fmt.Sprintf("sslcert='%s' ", sslcert)
	}
	if sslkey != "" {
		dsn += fmt.Sprintf("sslkey='%s' ", sslkey)
	}
	if sslrootcert != "" {
		dsn += fmt.Sprintf("sslrootcert='%s' ", sslrootcert)
	}
	if sslrootcert != "" {
		dsn += fmt.Sprintf("sslrootcert='%s' ", sslrootcert)
	}
	if timeout.Seconds() > 0 {
		dsn += fmt.Sprintf("connect_timeout=%.f ", timeout.Seconds())
	}

	return strings.TrimSpace(dsn)
}

func connect(dsn string) (*sql.DB, error) {
	// "open" in lib/pq just validates the provided dsn
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection to database with dsn %q: %v", dsn, err)
	}

	// dsn did validate — now try to actually reach the database
	if err := db.Ping(); err != nil { // this can take very long: lib/pq doesn't support context for Ping yet
		return nil, fmt.Errorf("failed to connect to database server: %v", err)
	}

	return db, nil
}
