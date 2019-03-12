package main

import (
	"flag"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// just define this here so we can run go test ./... from package root without warnings
var flagWithDb = flag.Bool("db", false, "Run tests against a test database.")

func Test_createDSN(t *testing.T) {
	host := "localhost"
	port := "5432"
	name := "app1"
	user := "mike"
	pass := "secret"
	sslmode := "verify-ca"
	sslcert := "certs/db.cert"
	sslkey := "certs/db.key"
	sslrootcert := "certs/ca.cert"
	timeout := time.Second * 42

	want := `host=localhost port=5432 dbname='app1' user='mike' password='secret' sslmode=verify-ca sslcert='certs/db.cert' sslkey='certs/db.key' sslrootcert='certs/ca.cert' sslrootcert='certs/ca.cert' connect_timeout=42`
	if got := createDSN(host, port, name, user, pass, sslmode, sslcert, sslkey, sslrootcert, timeout); got != want {
		t.Errorf("createDSN()\ngot  %q\nwant %q", got, want)
	}
	want = `password='ve ry$se\'cret!'`
	if got := createDSN("", "", "", "", `ve ry$se'cret!`, "", "", "", "", time.Second*0); got != want {
		t.Errorf("createDSN()\ngot  %q\nwant %q", got, want)
	}
}

func Test_connect(t *testing.T) {
	if *flagWithDb == false {
		t.Skip("skipping test: need a database")
	}
	dsn := "dbname=postgres user=postgres sslmode=disable connect_timeout=5"

	db, err := connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
}
