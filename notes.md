# migrathor provides SQL schema migration handling for PostgreSQL

## Dependencies

The library itself depends only on the *go standard libray* and uses `sql.DB` for its database operations.

## Environment

The core library by itself does not care about any env variables. You can pass environment vaues in the supporting command-line app. The command-line version of _migrathor_ uses [ff](github.com/peterbourgon/ff) under its hood and parses setting in exactly this order and priority:

* **flags** (overrides *config file*)
* **config file** (overrides *env variables*)
* **environment variables** (least priority)

## Transactions

PostgreSQL has transaction support for most DDL changes. _Migrathor_ takes advantage of this fact and runs every single migration in its own transaction. However, there are certain SQL commands which aren't supported within transactions (see this [list](#sql-commands-not-supported-within-transcations)).

_Migrathor_ takes a pragmatic approach for such occasions:

Insert the line `-- migrathor:no_transaction` at the very top of your migration file and _migrathor_ disables transaction support for that particular migration.

```sql
-- migrathor:no_transaction /* comments here are allowed */
VACUUM log;
CREATE xyz...;
```

> **PRO TIP**: Keep your migrations with transaction-unsupporting statements short. Split your statements into separate migrations and mark the one which doesn't support transactions with `-- migrathor:no_transaction`.

**Trivia**: In an earlier iteration, _migrathor_ tried to be clever and searched each migration with regular expressions for statements, which couldn't run within transactions, and if it found one, that whole migration ran without transaction support (other SQL migration tools like [flyway](https://flywaydb.org/) took this approach).

That course of action proved too brittle and limiting. The occurence of false positives and new/changed features in PostgreSQL showed that explicit switches within migration scripts were way more reliable.

### SQL commands not supported within transcations

If one tries to execute an unsupported statement within a transaction, PostgreSQL usually returns the error code `25001 (active_sql_transaction)`. The error message indicates the statement or keyword which isn't allowed within transactions.

List of common commands unsupported in transactions:

* `COMMIT PREPARED`
* `ROLLBACK PREPARED`
* `CLUSTER`
* `ALTER DATABASE SET TABLESPACE`
* `ALTER TYPE ... ADD VALUE`
* `ALTER SYSTEM`
* `DISCARD ALL`
* `CREATE SUBSCRIPTION`
* `DROP SUBSCRIPTION`
* `CREATE TABLESPACE`
* `DROP TABLESPACE`
* `CREATE DATABASE`
* `DROP DATABASE`
* `CREATE INDEX CONCURRENTLY`
* `DROP INDEX CONCURRENTLY`
* `REINDEX DATABASE`
* `REINDEX SCHEMA`
* `REINDEX SYSTEM`
* `VACUUM`

## What am I getting myself into

Nothing too serious :-) Having no external dependencies makes this library very lightweight — we mean to keep it that way. The gist of this package has been doing its work since 2014 (back then without `Context`) in multiple smaller and larger customer projects with several developers making changes against app databases.

This library:

* won't introduce external dependencies outside of the *go standard library* ever
* won't support *rollback applied migrations* behavior (see [why not](#why-no-down-operations))
* won't try to be *too clever* by parsing migrations or interfering with the migrations to be applied
* won't log anything anywhere unless you provide a custom logging destination
* won't swallow errors (sql-drivers often contain usefull data and we pass that along with our `DriverError` struct)
* won't panic (unless the panic bubbles up from the sql driver)

Instead this library aims to be:

* easy to read / digest
* pragmatic
* reliable

## Why no downgrade operations

### Reason 1

TODO(denis): See arguments of fwdb

### Reason 2

TODO(denis): Downmigrations are often "forgotten" when in deadline crunch mode (Mike goes back 5 steps, but Andy forgot down migrations for step 3: Mike is confused why everything seems off, wastes time searching, understanding up-migration of step 3, writing and testing down-migration: just use proper backup), less tested, won't cover every corner case

## License

The MIT License (MIT) — [Denis Brodbeck](https://github.com/denisbrodbeck). Please have a look at the [LICENSE](LICENSE) for more details.
