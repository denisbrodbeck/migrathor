// Package migrathor provides SQL schema migration handling for PostgreSQL.
//
//   https://github.com/denisbrodbeck/migrathor
//
// Features
//
// • simple and robust
//
// • forward-only migrations
//
// • PostgreSQL support only
//
// The focus of this package lies on simple, forward-only migrations supporting PostgreSQL only.
//
// This package has no external depencies and serves best when included as a library.
// No assumption is made on the provided PostgreSQL driver, the only dependency is `sql.DB`.
//
// That’s what it needs to do. That’s what it does. 🔕
package migrathor
