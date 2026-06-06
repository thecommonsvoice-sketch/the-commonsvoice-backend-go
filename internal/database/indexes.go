package database

import (
	"fmt"

	"gorm.io/gorm"
)

var articleIndexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_article_published_active
	 ON "Article" ("createdAt" DESC)
	 WHERE "status" = 'PUBLISHED' AND "deletedAt" IS NULL`,

	`CREATE INDEX IF NOT EXISTS idx_article_deleted_at
	 ON "Article" ("deletedAt")`,

	`CREATE INDEX IF NOT EXISTS idx_article_updated_at
	 ON "Article" ("updatedAt")`,
}

func EnsureIndexes(db *gorm.DB) error {
	for _, idx := range articleIndexes {
		if err := db.Exec(idx).Error; err != nil {
			return fmt.Errorf("index failed: %w\nSQL: %s", err, idx)
		}
	}
	return nil
}
