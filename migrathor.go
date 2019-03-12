package migrathor

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultHistoryTable    = "migrations"
	defaultTimestampFormat = "2006_01_02_150405"
)

// Logger is a generic logging func.
type Logger func(...interface{})

// FilenameFormatter takes a migration name and formats it into a filename.
type FilenameFormatter func(string) string

// A Migration implements database migrations with sql files using the native
// file system restricted to a specific directory tree.
//
// An empty Migration path is treated as ".".
type Migration struct {
	path      string
	table     string
	formatter FilenameFormatter
	logger    Logger
}

// New returns a new Migration.
func New(path string, options ...Option) *Migration {
	mig := &Migration{path: path}

	for _, option := range options {
		option(mig)
	}

	if mig.formatter == nil {
		mig.formatter = defaultFilenameFormatter
	}
	if mig.table == "" {
		mig.table = defaultHistoryTable
	}
	if mig.logger == nil {
		mig.logger = log.New(ioutil.Discard, "", 0).Print
	}

	return mig
}

// Create creates a new sql migration file in the migration directory.
//
//   cmd:     migration.Create("create_user_table")
//   created: database/migrations/2019_02_25_150455_create_user_table.sql
//   return:  2019_02_25_150455_create_user_table.sql
//
// The migration directory will be automatically created if it doesn't exist.
func (m *Migration) Create(name string) (filename string, err error) {
	file := m.formatter(name)

	if err := os.MkdirAll(m.path, 0755); err != nil {
		return "", fmt.Errorf("failed to create migrations directory %q: %v", m.path, err)
	}

	path := filepath.Join(m.path, file)
	header := fmt.Sprintf("/**\n* Name: %s\n* Date: %s\n*/\n\n", name, time.Now().Format(time.RFC3339))

	if err := ioutil.WriteFile(path, []byte(header), 0644); err != nil {
		return "", fmt.Errorf("failed to create migration at %q: %v", path, err)
	}

	return file, nil
}

func (m *Migration) Apply(ctx context.Context, db *sql.DB) (applied []string, err error) {
	exist, err := m.initialized(ctx, db)
	if err != nil {
		return []string{}, err
	}
	if !exist {
		if err := m.initialize(ctx, db); err != nil {
			return []string{}, err
		}
		m.logger("History table created successfully.")
	}

	available, err := m.available()
	if err != nil {
		return []string{}, err
	}

	applied, err = m.applied(ctx, db)
	if err != nil {
		return []string{}, err
	}

	// Are there available migrations which were not applied yet?
	pending := filterExcept(available, applied)
	if len(pending) == 0 {
		return []string{}, nil // nothing to do here
	}
	sort.Strings(pending)

	return m.apply(ctx, db, pending)
}

func (m *Migration) apply(ctx context.Context, db *sql.DB, pending []string) (applied []string, err error) {
	applied = []string{}
	insertCmd := fmt.Sprintf("INSERT INTO %s (migration, execution_time) VALUES ($1, $2);", m.table)
	// read pending migration files and execute them
	for _, migration := range pending {
		path := filepath.Join(m.path, migration)
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return applied, fmt.Errorf("failed to read file contents of %q: %v", path, err)
		}
		if txSupported(buf) {
			err = transaction(ctx, db, m.logger, func(tx *sql.Tx) error {
				// execute migration in transaction
				start := time.Now()
				if _, err := tx.ExecContext(ctx, string(buf)); err != nil {
					return &DriverError{"failed to execute SQL script " + path, err}
				}
				// log executed migration into history table
				if _, err := tx.ExecContext(ctx, insertCmd, migration, time.Since(start)); err != nil {
					return &DriverError{
						fmt.Sprintf("failed to execute SQL statement %q", strings.Join(strings.Fields(insertCmd), " ")),
						err,
					}
				}
				return nil
			})
			if err != nil {
				return applied, err
			}
		} else {
			// execute migration with no transaction support
			start := time.Now()
			if _, err := db.ExecContext(ctx, string(buf)); err != nil {
				return applied, &DriverError{"failed to execute SQL script " + path, err}
			}
			// log executed migration into history table
			if _, err := db.ExecContext(ctx, insertCmd, migration, time.Since(start)); err != nil {
				return applied, &DriverError{
					fmt.Sprintf("failed to execute SQL statement %q", strings.Join(strings.Fields(insertCmd), " ")),
					err,
				}
			}
		}
		applied = append(applied, migration)
	}

	return applied, nil
}

func (m *Migration) available() ([]string, error) {
	nodes, err := ioutil.ReadDir(m.path)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of migration files from %q: %v", m.path, err)
	}

	migrationFiles := []string{}
	for _, node := range nodes {
		if node.IsDir() {
			continue
		}
		if filepath.Ext(node.Name()) != ".sql" {
			continue
		}
		migrationFiles = append(migrationFiles, node.Name())
	}

	return migrationFiles, nil
}

// initialized returns whether the history table for applied migrations
// exists in the current schema.
func (m *Migration) initialized(ctx context.Context, db *sql.DB) (bool, error) {
	cmd := `
SELECT EXISTS (
	SELECT 1
	FROM pg_tables
	WHERE schemaname = current_schema()
	AND tablename = $1
);`[1:]
	row := db.QueryRowContext(ctx, cmd, m.table)

	var exist bool
	if err := row.Scan(&exist); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, &DriverError{"failed to verify existence of history table", err}
	}

	return exist, nil
}

// initialize creates the history table
// which keeps track of all applied migrations.
func (m *Migration) initialize(ctx context.Context, db *sql.DB) error {
	stmt := `
CREATE TABLE IF NOT EXISTS %s (
	id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	migration TEXT NOT NULL UNIQUE,
	applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
	execution_time REAL NOT NULL
);`[1:]
	cmd := fmt.Sprintf(stmt, m.table)

	return transaction(ctx, db, m.logger, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, cmd); err != nil {
			return &DriverError{"failed to create history table", err}
		}
		return nil
	})
}

// applied returns all completed migrations from the history table.
func (m *Migration) applied(ctx context.Context, db *sql.DB) ([]string, error) {
	cmd := fmt.Sprintf(`SELECT migration FROM %s ORDER BY id ASC;`, m.table)
	rows, err := db.QueryContext(ctx, cmd)
	if err != nil {
		return nil, &DriverError{"failed to query applied migrations", err}
	}
	defer logCloser(rows, m.logger)

	applied := []string{}
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, &DriverError{"failed to row scan entry in query for applied migrations", err}
		}
		applied = append(applied, m)
	}

	if err := rows.Err(); err != nil {
		return nil, &DriverError{"failed to query applied migrations", err}
	}

	return applied, nil
}

// transaction is a utility function to execute SQL inside a transaction
//
// see: https://stackoverflow.com/a/23502629
func transaction(ctx context.Context, db *sql.DB, logger Logger, txFunc func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return &DriverError{"failed to begin db transaction", err}
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				logger(err)
			}
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			// err is non-nil; don't change it
			if err := tx.Rollback(); err != nil {
				logger(err)
			}
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
		}
	}()

	err = txFunc(tx)

	return err
}

// txSupported checks whether input is prefixed with `-- migrathor:no_transaction`
//
// Returns true if input has no transaction suppressor flag in first line.
func txSupported(s []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(s), []byte(`-- migrathor:no_transaction`)) == false
}

// logCloser is a convenience logger for deferred execution.
//
// This fuction takes any struct implementing the io.Closer interface and closes
// it it upon execution. Errors get logged to the provided logger.
//
//   file, _ := os.Open("some/file")
//   defer logCloser(file, logger)
func logCloser(c io.Closer, logger Logger) {
	if err := c.Close(); err != nil {
		logger("failed to close handle: " + err.Error())
	}
}

// Option controls some aspects of migration behavior.
type Option func(*Migration)

// WithHistoryTable tells New to use the provided name as default history table
// name for applied migrations.
func WithHistoryTable(name string) Option {
	return func(c *Migration) {
		c.table = name
	}
}

// WithLogger tells New to use the provided logger for internal logging.
func WithLogger(logger Logger) Option {
	return func(c *Migration) {
		c.logger = logger
	}
}

// WithFilenameFormatter tells New to use the provided logger for internal logging.
func WithFilenameFormatter(formatter FilenameFormatter) Option {
	return func(c *Migration) {
		c.formatter = formatter
	}
}

func defaultFilenameFormatter(name string) string {
	file := time.Now().UTC().Format(defaultTimestampFormat) + "_" + strings.ToLower(name)
	if filepath.Ext(file) != ".sql" {
		file += ".sql"
	}
	return file
}
