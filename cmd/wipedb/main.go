package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	_ "github.com/jackc/pgx/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("unexpected error: %s", err)
	}

	log.Print("database successfuly wiped")
}

func run() error {
	args := os.Args
	if len(args) != 2 {
		return errors.New("database URI expected as the only argument")
	}

	uri := args[1]

	db, err := sql.Open("pgx", uri)
	if err != nil {
		return fmt.Errorf("cannot open database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tables, err := collectTables(ctx, db)
	if err != nil {
		return fmt.Errorf("cannot collect schemas: %w", err)
	}

	for _, tableName := range tables {
		_, err = db.ExecContext(ctx, `DROP TABLE `+tableName)
		if err != nil {
			return fmt.Errorf("cannot perform table wipe: %w", err)
		}

		log.Printf("table '%s' has been successfully wiped", tableName)
	}

	return nil
}

var reservedSchemas = []string{"pg_catalog", "information_schema"}

func collectTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT table_schema, table_name FROM information_schema.tables`)
	if err != nil {
		return nil, fmt.Errorf("cannot perform query: %w", err)
	}

	defer rows.Close()

	var tables []string
	for rows.Next() {
		var schemaName, tableName string

		if err := rows.Scan(&schemaName, &tableName); err != nil {
			return nil, fmt.Errorf("cannot scan row: %w", err)
		}

		if !slices.Contains(reservedSchemas, schemaName) {
			tables = append(tables, schemaName+"."+tableName)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error while performing rows scan: %w", err)
	}

	return tables, nil
}
