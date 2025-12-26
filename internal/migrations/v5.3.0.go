package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_3_0 performs the DB migrations.
func V5_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Create webhook_status enum.
	_, err := db.Exec(`
		DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'webhook_status') THEN
				CREATE TYPE webhook_status AS ENUM ('enabled', 'disabled');
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	// Create webhook_log_status enum.
	_, err = db.Exec(`
		DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'webhook_log_status') THEN
				CREATE TYPE webhook_log_status AS ENUM ('pending', 'success', 'failed');
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	// Create webhooks table.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id               SERIAL PRIMARY KEY,
			uuid             uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
			name             TEXT NOT NULL,
			url              TEXT NOT NULL,
			status           webhook_status NOT NULL DEFAULT 'enabled',
			events           TEXT[] NOT NULL DEFAULT '{}',
			auth_type        TEXT NOT NULL DEFAULT 'none',
			auth_basic_user  TEXT NOT NULL DEFAULT '',
			auth_basic_pass  TEXT NOT NULL DEFAULT '',
			auth_hmac_secret TEXT NOT NULL DEFAULT '',
			max_retries      INTEGER NOT NULL DEFAULT 3,
			retry_interval   TEXT NOT NULL DEFAULT '30s',
			timeout          TEXT NOT NULL DEFAULT '30s',
			created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_webhooks_status ON webhooks(status);
		CREATE INDEX IF NOT EXISTS idx_webhooks_events ON webhooks USING GIN(events);
	`)
	if err != nil {
		return err
	}

	// Create webhook_logs table.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS webhook_logs (
			id            BIGSERIAL PRIMARY KEY,
			webhook_id    INTEGER NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE ON UPDATE CASCADE,
			event         TEXT NOT NULL,
			url           TEXT NOT NULL,
			payload       JSONB NOT NULL DEFAULT '{}',
			status        webhook_log_status NOT NULL DEFAULT 'pending',
			response_code INTEGER NULL,
			response_body TEXT NOT NULL DEFAULT '',
			error         TEXT NOT NULL DEFAULT '',
			attempts      INTEGER NOT NULL DEFAULT 0,
			next_retry_at TIMESTAMP WITH TIME ZONE NULL,
			created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_webhook_id ON webhook_logs(webhook_id);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_status ON webhook_logs(status);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_next_retry ON webhook_logs(next_retry_at) WHERE status = 'pending';
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_created_at ON webhook_logs(created_at);
	`)
	if err != nil {
		return err
	}

	return nil
}
