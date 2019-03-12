package migrathor_test

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/denisbrodbeck/migrathor"
)

func ExampleWithFilenameFormatter() {
	// prefer unix time stamps instead of default datetime formatter
	fn := func(name string) string {
		return fmt.Sprintf("%d_%s.sql", time.Now().Unix(), strings.ToLower(name))
	}
	_ = migrathor.New("database/migrations", migrathor.WithFilenameFormatter(fn))
	// filename, _ := migration.Create("Create_Users_Table")
	// Would Output: 1552392673_create_users_table.sql
}

func ExampleWithLogger() {
	logger := log.New(os.Stdout, "", 0).Print
	_ = migrathor.New("database/migrations", migrathor.WithLogger(logger))
}

func ExampleWithHistoryTable() {
	table := "schema_history"
	_ = migrathor.New("database/migrations", migrathor.WithHistoryTable(table))
}
