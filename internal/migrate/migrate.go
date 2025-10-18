package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultDir = "migrations"

func init() {
	goose.SetLogger(log.New(os.Stdout, "[goose] ", log.LstdFlags))
	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}
}

func dir(path string) string {
	if path != "" {
		return path
	}
	return defaultDir
}

func Up(ctx context.Context, dsn string) error {
	return UpDir(ctx, dsn, "")
}

func UpDir(ctx context.Context, dsn, migrationsPath string) error {
	return withDB(ctx, dsn, func(ctx context.Context, db *sql.DB) error {
		return goose.UpContext(ctx, db, dir(migrationsPath))
	})
}

func Down(ctx context.Context, dsn string, steps int) error {
	return DownDir(ctx, dsn, "", steps)
}

func DownDir(ctx context.Context, dsn, migrationsPath string, steps int) error {
	if steps <= 0 {
		steps = 1
	}
	return withDB(ctx, dsn, func(ctx context.Context, db *sql.DB) error {
		path := dir(migrationsPath)
		for i := 0; i < steps; i++ {
			if err := goose.DownContext(ctx, db, path); err != nil {
				return err
			}
		}
		return nil
	})
}

func Status(ctx context.Context, dsn string) error {
	return StatusDir(ctx, dsn, "")
}

func StatusDir(ctx context.Context, dsn, migrationsPath string) error {
	return withDB(ctx, dsn, func(ctx context.Context, db *sql.DB) error {
		return goose.StatusContext(ctx, db, dir(migrationsPath))
	})
}

func Version(ctx context.Context, dsn string) (int64, error) {
	return VersionDir(ctx, dsn, "")
}

func VersionDir(ctx context.Context, dsn, migrationsPath string) (int64, error) {
	var version int64
	err := withDB(ctx, dsn, func(ctx context.Context, db *sql.DB) error {
		v, err := goose.GetDBVersion(db)
		if err != nil {
			return err
		}
		version = v
		return nil
	})
	return version, err
}

func Redo(ctx context.Context, dsn string) error {
	return RedoDir(ctx, dsn, "")
}

func RedoDir(ctx context.Context, dsn, migrationsPath string) error {
	return withDB(ctx, dsn, func(ctx context.Context, db *sql.DB) error {
		return goose.RedoContext(ctx, db, dir(migrationsPath))
	})
}

func withDB(ctx context.Context, dsn string, fn func(context.Context, *sql.DB) error) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	if _, err := goose.EnsureDBVersion(db); err != nil {
		if strings.Contains(err.Error(), "goose_db_version") {
			if err := createGooseVersionTable(ctx, db); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("ensure goose version table: %w", err)
		}
	}

	return fn(ctx, db)
}

func ParseSteps(arg string) (int, error) {
	if arg == "" {
		return 1, nil
	}
	n, err := strconv.Atoi(arg)
	if err != nil {
		return 0, errors.New("steps must be a positive integer")
	}
	if n <= 0 {
		return 0, errors.New("steps must be greater than zero")
	}
	return n, nil
}

func ContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}

func ResolveDir(custom string) (string, error) {
	path := dir(custom)
	if filepath.IsAbs(path) {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, path), nil
}

func createGooseVersionTable(ctx context.Context, db *sql.DB) error {
	const createTable = `CREATE TABLE IF NOT EXISTS goose_db_version (
  id SERIAL PRIMARY KEY,
  version_id BIGINT NOT NULL,
  is_applied BOOLEAN NOT NULL,
  tstamp TIMESTAMP NOT NULL DEFAULT now()
)`
	if _, err := db.ExecContext(ctx, createTable); err != nil {
		return fmt.Errorf("create goose version table: %w", err)
	}
	const insertRow = `INSERT INTO goose_db_version (version_id, is_applied)
SELECT 0, TRUE
WHERE NOT EXISTS (SELECT 1 FROM goose_db_version)`
	if _, err := db.ExecContext(ctx, insertRow); err != nil {
		return fmt.Errorf("init goose version table: %w", err)
	}
	return nil
}
