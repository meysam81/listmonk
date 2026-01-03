package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_0_0 performs the DB migrations for adding MJML content type.
func V7_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Add 'mjml' to content_type enum.
	_, err := db.Exec(`
		DO $$ BEGIN
			-- Add 'mjml' to content_type enum if it doesn't exist
			IF NOT EXISTS (
				SELECT 1 FROM pg_enum
				WHERE enumlabel = 'mjml'
				AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'content_type')
			) THEN
				ALTER TYPE content_type ADD VALUE 'mjml';
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	return nil
}
