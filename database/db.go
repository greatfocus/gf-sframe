package database

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
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
func (c *Conn) sslCertificate() (cert, key, ca string) {
	var root string
	key = c.createSSLCert("client.key", os.Getenv("DB_SSL_KEY"))
	cert = c.createSSLCert("client.crt", os.Getenv("DB_SSL_CERT"))
	sslrootcert := os.Getenv("DB_ROOT_CA")
	if sslrootcert != "" {
		root = c.createSSLCert("ca.crt", os.Getenv("DB_ROOT_CA"))
	}
	return key, cert, root
}

// createSSLCert makes cert in image
func (c *Conn) createSSLCert(filename string, content string) string {
	var path = os.Getenv("APP_PATH") + "/" + filename
	path = filepath.Clean(path)
	_, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			_ = file.Close()
		}()
		err = os.Chmod(path, 0600)
		if err != nil {
			log.Fatal(err)
		}

		cnt, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			file.WriteString(content)
		}
		file.WriteString(string(cnt))
	}

	return path
}

// Connect method make a database connection
func (c *Conn) connect() {
	// initialize variables rom config
	log.Println("Preparing Database configuration")
	host := os.Getenv("DB_HOST")
	database := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	sslmode := "require"
	sslkey := os.Getenv("DB_SSL_KEY")
	sslcert := os.Getenv("DB_SSL_CERT")
	var sslkeyPath, sslcertPath, sslcaPath string

	// prepare ssl connection files
	if sslkey != "" && sslcert != "" {
		sslkeyPath, sslcertPath, sslcaPath = c.sslCertificate()
	}

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
	var psqlInfo string
	if sslcaPath != "" {
		sslmode = "verify-full"
		psqlInfo = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s sslkey=%s sslcert=%s",
			host, port, user, password, database, sslmode, sslcaPath, sslkeyPath, sslcertPath)
	} else {
		psqlInfo = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s sslkey=%s sslcert=%s",
			host, port, user, password, database, sslmode, sslkeyPath, sslcertPath)
	}

	conn, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	// confirm connection
	err = conn.Ping()
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
func (c *Conn) Insert(ctx context.Context, query string, args ...interface{}) (int64, bool) {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return 0, false
	}
	defer func() {
		_ = stmt.Close()
	}()
	res, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return 0, false
	}
	rows, err := res.RowsAffected()
	if err != nil || rows < 1 {
		return 0, false
	}
	return rows, true
}

// Query method make a resultset rows query to the databases
func (c *Conn) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return &sql.Rows{}, err
	}
	defer func() {
		_ = stmt.Close()
	}()
	rows, err := stmt.QueryContext(ctx, args...)
	defer func() {
		_ = rows.Close()
	}()
	return rows, err
}

// Select method make a single row query to the databases
func (c *Conn) Select(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return &sql.Row{}
	}
	defer func() {
		_ = stmt.Close()
	}()
	rows := stmt.QueryRowContext(ctx, args...)
	return rows
}

// Update method executes update database changes to the databases
func (c *Conn) Update(ctx context.Context, query string, args ...interface{}) bool {
	return updateOrDelete(c, query, ctx, args)
}

// Delete method executes delete database changes to the databases
func (c *Conn) Delete(ctx context.Context, query string, args ...interface{}) bool {
	return updateOrDelete(c, query, ctx, args)
}

// update or delete records
func updateOrDelete(c *Conn, query string, ctx context.Context, args []interface{}) bool {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		return false
	}
	defer func() {
		_ = stmt.Close()
	}()
	res, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return false
	}

	count, err := res.RowsAffected()
	if err != nil || count < 1 {
		return false
	}
	return true
}
