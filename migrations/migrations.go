package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Run(db *sql.DB, dir string) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migration files: %w", err)
	}

	sort.Strings(files)

	for _, file := range files {
		version := filepath.Base(file)

		applied, err := isApplied(db, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := runMigrationFile(db, file); err != nil {
			return fmt.Errorf("run migration %s: %w", version, err)
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}

	return nil
}

func isApplied(db *sql.DB, version string) (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&count); err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return count > 0, nil
}

func runMigrationFile(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := strings.Split(string(content), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := execStatement(tx, stmt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func execStatement(tx *sql.Tx, stmt string) error {
	if shouldSkipAddColumn(tx, stmt) {
		return nil
	}

	_, err := tx.Exec(stmt)
	return err
}

func shouldSkipAddColumn(tx *sql.Tx, stmt string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(stmt), " "))

	if !strings.HasPrefix(normalized, "alter table ") || !strings.Contains(normalized, " add column ") {
		return false
	}

	parts := strings.Fields(stmt)
	if len(parts) < 6 {
		return false
	}

	tableName := strings.Trim(parts[2], "`\"[]")
	columnName := strings.Trim(parts[5], "`\"[]")

	exists, err := columnExists(tx, tableName, columnName)
	if err != nil {
		return false
	}

	return exists
}

func columnExists(tx *sql.Tx, tableName, columnName string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)

		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}

		if strings.EqualFold(name, columnName) {
			return true, nil
		}
	}

	return false, rows.Err()
}
