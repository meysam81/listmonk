package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_1_0 performs the DB migrations for adding MJML content type.
func V6_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Applying v6.1.0 migrations...")
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
