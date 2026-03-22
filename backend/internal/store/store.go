package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type DB struct {
	*bun.DB
	Project uuid.UUID
}

func Connect(dsn string, projectUUID string) (*DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	sqldb.SetMaxOpenConns(10)
	b := bun.NewDB(sqldb, pgdialect.New())
	pu, err := uuid.Parse(projectUUID)
	if err != nil {
		return nil, fmt.Errorf("project uuid: %w", err)
	}
	return &DB{DB: b, Project: pu}, nil
}

func (db *DB) Migrate(ctx context.Context) error {
	_, err := db.ExecContext(ctx, migrationSQL)
	return err
}

const migrationSQL = `
DO $$ BEGIN
    CREATE TYPE role_type_enum AS ENUM (
        'norole', 'system', 'assistant', 'user', 'function', 'tool'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS users (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    id BIGSERIAL,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    updated_at timestamptz NOT NULL DEFAULT current_timestamp,
    deleted_at timestamptz,
    user_id VARCHAR NOT NULL,
    email VARCHAR,
    first_name VARCHAR,
    last_name VARCHAR,
    project_uuid uuid NOT NULL,
    metadata jsonb,
    PRIMARY KEY (uuid),
    UNIQUE (user_id)
);

CREATE TABLE IF NOT EXISTS sessions (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    id BIGSERIAL,
    session_id VARCHAR NOT NULL,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    updated_at timestamptz NOT NULL DEFAULT current_timestamp,
    deleted_at timestamptz,
    ended_at timestamptz,
    metadata jsonb,
    user_id VARCHAR,
    project_uuid uuid NOT NULL,
    PRIMARY KEY (uuid),
    UNIQUE (session_id),
    FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS messages (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    id BIGSERIAL,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    updated_at timestamptz,
    deleted_at timestamptz,
    session_id VARCHAR NOT NULL,
    project_uuid uuid NOT NULL,
    role VARCHAR NOT NULL,
    role_type role_type_enum DEFAULT 'norole',
    content VARCHAR NOT NULL,
    token_count BIGINT NOT NULL DEFAULT 0,
    metadata jsonb,
    name VARCHAR DEFAULT '',
    processed boolean,
    PRIMARY KEY (uuid),
    FOREIGN KEY (session_id) REFERENCES sessions (session_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS user_user_id_idx ON users (user_id);
CREATE INDEX IF NOT EXISTS memstore_session_id_idx ON messages (session_id);
CREATE INDEX IF NOT EXISTS session_user_id_idx ON sessions (user_id);

CREATE TABLE IF NOT EXISTS context_templates (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    content TEXT NOT NULL,
    project_uuid uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    updated_at timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (uuid),
    UNIQUE (id)
);

CREATE TABLE IF NOT EXISTS graphs (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    graph_id VARCHAR NOT NULL,
    user_id VARCHAR,
    project_uuid uuid NOT NULL,
    metadata jsonb,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    updated_at timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (uuid),
    UNIQUE (graph_id)
);

CREATE TABLE IF NOT EXISTS tasks (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    task_id VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    progress double precision NOT NULL DEFAULT 0,
    error TEXT,
    project_uuid uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    updated_at timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (uuid),
    UNIQUE (task_id)
);

CREATE TABLE IF NOT EXISTS custom_instructions (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    name VARCHAR NOT NULL,
    text TEXT NOT NULL,
    scope VARCHAR NOT NULL,
    scope_id VARCHAR,
    project_uuid uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (uuid)
);

CREATE TABLE IF NOT EXISTS user_summary_instructions (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    name VARCHAR NOT NULL,
    text TEXT NOT NULL,
    project_uuid uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (uuid)
);

CREATE TABLE IF NOT EXISTS entity_types (
    uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    project_uuid uuid NOT NULL,
    payload jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY (uuid),
    UNIQUE (project_uuid)
);
`
