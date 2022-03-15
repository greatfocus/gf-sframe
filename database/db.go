package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Conn struct
type Conn struct {
	conn *sql.DB
}

// Init database connection
func (c *Conn) Init() {
	c.connect()
}

// Connect method make a database connection
func (c *Conn) connect() {
	// initialize variables rom config
	log.Println("Preparing Database configuration")
	host := os.Getenv("DB_HOST")
	database := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	sslmode := "verify-full"
	sslrootcert := os.Getenv("DB_ROOT+CA")
	sslkey := os.Getenv("DB_SSL_KEY")
	sslcert := os.Getenv("DB_SSL_CERT")

	port, err := strconv.ParseUint(os.Getenv("DB_PORT"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	maxLifetimeVal, err := strconv.ParseUint(os.Getenv("DB_MaxLifetime"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}
	maxLifetime := time.Duration(maxLifetimeVal) * time.Minute

	maxIdleConns, err := strconv.ParseInt(os.Getenv("DB_MaxIdleConns"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	maxOpenConns, err := strconv.ParseInt(os.Getenv("DB_MaxOpenConns"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	// create database connection
	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s sslkey=%s sslcert=%s",
		host, port, user, password, database, sslmode, sslrootcert, sslkey, sslcert)
	conn, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}
	conn.SetConnMaxLifetime(maxLifetime)
	conn.SetMaxIdleConns(int(maxIdleConns))
	conn.SetMaxOpenConns(int(maxOpenConns))
	log.Println("Initiating Database connection")

	// execute database scripts
	c.executeSchema(conn)
	c.RebuildIndexes(conn, database)
	c.conn = conn
}

// ExecuteSchema prepare and execute database changes
func (c *Conn) executeSchema(db *sql.DB) {
	// read the scripts in the folder
	var path = os.Getenv("APP_PATH") + "/database/"
	path = filepath.Clean(path)
	log.Println("Preparing to execute database schema")
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	// loop thru files to create schemas
	for _, f := range files {
		var schema = path + "/" + f.Name()
		schemaPath := filepath.Clean(schema)
		scriptFile, err := os.OpenFile(schemaPath, os.O_RDONLY, 0600)
		if err != nil {
			log.Fatal(fmt.Println(err))
		}
		// read the config file
		scriptContent, err := ioutil.ReadAll(scriptFile)
		if err != nil {
			log.Fatal(fmt.Println(err))
		}
		sql := string(scriptContent)
		log.Println("Executing schema: ", schemaPath)
		if _, err := db.Exec(sql); err != nil {
			log.Fatal(fmt.Println(err))
		}
	}

	log.Println("Database scripts successfully executed")
}

// RebuildIndexes within sframe
func (c *Conn) RebuildIndexes(db *sql.DB, dbname string) {
	log.Println("Rebuild Indexes")

	// Rebuild Indexes
	sqlReindexScript := string("REINDEX DATABASE " + dbname + ";")
	if _, err := db.Exec(sqlReindexScript); err != nil {
		log.Fatal(fmt.Println(err))
	}

	log.Println("Rebuild Indexes successfully executed")
}

// Insert method make a single row query to the databases
func (c *Conn) Insert(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.conn.QueryRowContext(ctx, query, args...)
}

// Query method make a resultset rows query to the databases
func (c *Conn) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.conn.QueryContext(ctx, query, args...)
}

// Select method make a single row query to the databases
func (c *Conn) Select(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.conn.QueryRowContext(ctx, query, args...)
}

// Update method executes update database changes to the databases
func (c *Conn) Update(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.conn.ExecContext(ctx, query, args...)
}

// Delete method executes delete database changes to the databases
func (c *Conn) Delete(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.conn.ExecContext(ctx, query, args...)
}
