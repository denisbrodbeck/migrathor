package migrathor

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var (
	flagWithDb = flag.Bool("db", false, "Run tests against a test database.")
	database   *sql.DB // populated by TestMain when using flag -with-db
	nullLogger = log.New(ioutil.Discard, "", 0).Print
)

func TestMain(m *testing.M) {
	flag.Parse()

	if *flagWithDb {
		db, err := connect()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer db.Close()
		database = db // make db globally available
	}

	os.Exit(m.Run())
}

func connect() (*sql.DB, error) {
	dsn := "dbname=postgres user=postgres sslmode=disable connect_timeout=5"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping test database: %v", err)
	}

	return db, nil
}

func cleanup(ctx context.Context, db *sql.DB, t *testing.T) {
	cmd := `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`

	err := transaction(ctx, db, nullLogger, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, cmd); err != nil {
			return &DriverError{"failed to reset public test schema", err}
		}
		return nil
	})
	if err != nil {
		t.Errorf("%+v", err)
	}
}

func Test_txSupported(t *testing.T) {
	tests := []string{
		"-- migrathor:no_transaction\nVACUUM log;",
		"  \n  -- migrathor:no_transaction",
		"  -- migrathor:no_transaction    /* comment */",
		"-- migrathor:no_transaction",
	}
	for _, tt := range tests {
		if got := txSupported([]byte(tt)); got {
			t.Errorf("txSupported(): transaction suppressor did not fire: %s", tt)
		}
	}
	tests = []string{
		"CREATE TABLE user();",
		"CREATE TABLE user();\nDO DATABASE STUFF();",
		"RANDOM ACCESS MEMORY;\n  -- migrathor:no_transaction /* suppressor not on first line */",
	}
	for _, tt := range tests {
		if got := txSupported([]byte(tt)); got == false {
			t.Errorf("txSupported(): transaction suppressor should not trigger on: %s", tt)
		}
	}
}

func TestMigration_Create(t *testing.T) {
	dir, err := ioutil.TempDir("", "migrathor_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	migration := New(dir)
	name, err := migration.Create("stuff")
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().IsRegular() == false {
		if err != nil {
			t.Errorf("created migration file should be regular file, is not: %s", fi.Mode())
		}
	}
}

func TestMigration_initialize(t *testing.T) {
	if *flagWithDb == false {
		t.Skip("skipping test: need a database")
	}
	ctx := context.Background()
	defer cleanup(ctx, database, t)

	migration := New("testdata")

	exist, err := migration.initialized(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if exist == true {
		t.Fatal("history table shouldn't exist but does")
	}

	// init history table
	if err := migration.initialize(ctx, database); err != nil {
		t.Fatal(err)
	}

	exist, err = migration.initialized(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if exist == false {
		t.Fatal("history table should exist but does not")
	}
}

func TestApply(t *testing.T) {
	if *flagWithDb == false {
		t.Skip("skipping test: need a database")
	}

	ctx := context.Background()
	defer cleanup(ctx, database, t)

	migration := New("testdata")

	// apply initial migrations
	got, err := migration.Apply(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2019_03_05_173612_create_users.sql", "2019_03_05_213554_add_users.sql"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Apply()\ngot  %v\nwant %v\n", got, want)
	}

	// add another migration
	name := time.Now().Format(defaultTimestampFormat + "_create_log.sql")
	file := filepath.Join("testdata", name)
	sql := `CREATE TABLE log (id bigserial PRIMARY KEY, msg TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());`
	if err := ioutil.WriteFile(file, []byte(sql), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)
	// apply added migration
	got, err = migration.Apply(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	want = []string{name}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Apply()\ngot  %v\nwant %v\n", got, want)
	}

	// add another migration with no transaction support
	name = time.Now().Format(defaultTimestampFormat + "_vacuum_log.sql")
	file2 := filepath.Join("testdata", name)
	sql = "-- migrathor:no_transaction\nVACUUM log;"
	if err := ioutil.WriteFile(file2, []byte(sql), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file2)
	// apply added migration
	got, err = migration.Apply(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	want = []string{name}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Apply()\ngot  %v\nwant %v\n", got, want)
	}
}

func TestApplyInvalid(t *testing.T) {
	if *flagWithDb == false {
		t.Skip("skipping test: need a database")
	}

	ctx := context.Background()
	defer cleanup(ctx, database, t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	migration := New(dir)

	// add broken migration
	file := filepath.Join(dir, time.Now().Format(defaultTimestampFormat+"_create_log.sql"))
	if err := ioutil.WriteFile(file, []byte(`CREATE TABLEtypo log ();`), 0644); err != nil {
		t.Fatal(err)
	}
	// apply broken migration
	_, err = migration.Apply(ctx, database)
	if err == nil {
		t.Fatal("apply returned no error, should have because of invalid sql")
	}

	// add broken migration without transaction support
	if err := ioutil.WriteFile(file, []byte(`VACUUM notthere;`), 0644); err != nil {
		t.Fatal(err)
	}
	// apply broken migration without transaction
	_, err = migration.Apply(ctx, database)
	if err == nil {
		t.Fatal("apply returned no error, should have because of invalid sql")
	}
}

func Test_transaction(t *testing.T) {
	if *flagWithDb == false {
		t.Skip("skipping test: need a database")
	}

	ctx := context.Background()
	defer cleanup(ctx, database, t)

	cmd := `CREATE TABLEups data();`

	err := transaction(ctx, database, nullLogger, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, cmd); err != nil {
			return &DriverError{"wrong syntax", err}
		}
		return nil
	})
	if err == nil {
		t.Fatal("err is nil, should not")
	}
}

func Test_transactionPanic(t *testing.T) {
	if *flagWithDb == false {
		t.Skip("skipping test: need a database")
	}

	defer func() {
		if p := recover(); p == nil {
			t.Fatal("should have panicked, did not")
		}
	}()

	transaction(context.Background(), database, nullLogger, func(tx *sql.Tx) error {
		panic("random access memory")
	})
}

func TestMigration_available(t *testing.T) {
	migration := New("testdata")

	got, err := migration.available()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2019_03_05_173612_create_users.sql", "2019_03_05_213554_add_users.sql"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("available()\ngot  %v\nwant %v\n", got, want)
	}

	migration = New("doesnotexist")
	_, err = migration.available()
	if err == nil {
		t.Fatal("wanted does not exist err, got nil")
	}
}

type closer struct{}

func (closer) Close() error {
	return fmt.Errorf("not closed")
}

func Test_logCloser(t *testing.T) {
	c := &closer{}
	got := ""
	l := func(a ...interface{}) {
		got = fmt.Sprint(a)
	}
	logCloser(c, l)
	want := "[failed to close handle: not closed]"
	if got != want {
		t.Errorf("wrong logCloser output: got %s, want %s", got, want)
	}
}

func TestNew(t *testing.T) {
	table := "history"
	logger := nullLogger
	formatter := func(name string) string {
		return fmt.Sprintf("%d_%s.sql", time.Now().Unix(), strings.ToLower(name))
	}
	_ = New("testdata", WithHistoryTable(table), WithFilenameFormatter(formatter), WithLogger(logger))
}
