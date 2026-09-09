package sloop

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

type LocalPaths struct {
	Root    string
	DB      string
	Objects string
	Cache   string
}

func PathsForProject(projectID string) (LocalPaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return LocalPaths{}, fmt.Errorf("locate user home: %w", err)
	}
	var base string
	if runtime.GOOS == "darwin" {
		base = filepath.Join(home, "Library", "Application Support")
	} else {
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
	}
	root := filepath.Join(base, "sloop", projectID)
	return LocalPaths{
		Root:    root,
		DB:      filepath.Join(root, "sloop.db"),
		Objects: filepath.Join(root, "objects"),
		Cache:   filepath.Join(root, "cache"),
	}, nil
}

type Store struct {
	DB    *sql.DB
	Paths LocalPaths
}

func OpenStore(projectID string) (*Store, error) {
	paths, err := PathsForProject(projectID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(paths.Objects, "sha256"), 0o755); err != nil {
		return nil, fmt.Errorf("create object store: %w", err)
	}
	if err := os.MkdirAll(paths.Cache, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	db, err := sql.Open("sqlite", paths.DB)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("configure database: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{DB: db, Paths: paths}, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS specifications (
    uuid TEXT PRIMARY KEY,
    id TEXT NOT NULL UNIQUE,
    number INTEGER NOT NULL UNIQUE,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    body TEXT NOT NULL,
    parents_json TEXT NOT NULL,
    features_json TEXT NOT NULL DEFAULT '{}',
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_agent INTEGER NOT NULL,
    updated_at TEXT NOT NULL,
    head_hash TEXT,
    dirty INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS revision_index (
    hash TEXT PRIMARY KEY,
    spec_uuid TEXT NOT NULL,
    spec_id TEXT NOT NULL,
    revision_number INTEGER NOT NULL,
    status TEXT NOT NULL,
    title TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_agent INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    object_path TEXT NOT NULL,
    UNIQUE(spec_uuid, revision_number)
);
CREATE INDEX IF NOT EXISTS revisions_by_spec ON revision_index(spec_uuid, revision_number);
CREATE TABLE IF NOT EXISTS revision_parents (
    revision_hash TEXT NOT NULL,
    parent_hash TEXT NOT NULL,
    PRIMARY KEY(revision_hash, parent_hash)
);
CREATE TABLE IF NOT EXISTS specification_references (
    id TEXT PRIMARY KEY,
    spec_uuid TEXT NOT NULL,
    kind TEXT NOT NULL,
    path TEXT NOT NULL,
    start_line INTEGER,
    end_line INTEGER,
    git_commit TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS references_by_spec ON specification_references(spec_uuid);
CREATE TABLE IF NOT EXISTS reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spec_uuid TEXT NOT NULL,
    revision_hash TEXT NOT NULL,
    result TEXT NOT NULL,
    reason TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_agent INTEGER NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS agent_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spec_uuid TEXT NOT NULL,
    revision_hash TEXT NOT NULL,
    result TEXT NOT NULL,
    reason TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS status_transitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spec_uuid TEXT NOT NULL,
    based_on_revision TEXT,
    resulting_revision TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_agent INTEGER NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS git_relations (
    spec_uuid TEXT NOT NULL,
    revision_hash TEXT NOT NULL,
    git_commit TEXT NOT NULL,
    relation TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY(spec_uuid, revision_hash, git_commit, relation)
);
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	if err := ensureColumn(db, "specifications", "features_json", "TEXT NOT NULL DEFAULT '{}'"); err != nil {
		return err
	}
	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return fmt.Errorf("inspect %s schema: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("inspect %s schema: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect %s schema: %w", table, err)
	}
	if _, err := db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition); err != nil {
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}
