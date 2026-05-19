package database

import (
	"fmt"

	"gorm.io/gorm"
)

// PreMigrate 在 GORM AutoMigrate 之前执行的 DDL
// 主要是列改名 / 列删除 -- AutoMigrate 不会处理这两种变更,且对已有数据敏感
// 必须在 AutoMigrate 之前,否则 GORM 看到 Go 模型上的新字段会尝试 ADD COLUMN,
// 而对 NOT NULL 列加列时,旧行的 NULL 会触发约束错误
func PreMigrate(db *gorm.DB) error {
	statements := []string{
		// 一次性迁移:把 file_id 列改名为 bot_file_id(对应 model.File.BotFileID)
		// 仅当旧列还存在时执行,幂等
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'files' AND column_name = 'file_id'
			) THEN
				ALTER TABLE files RENAME COLUMN file_id TO bot_file_id;
			END IF;
		END $$`,

		// 一次性迁移:删除已废弃的 thumbnail_file_id 列(代码层从未读取)
		`ALTER TABLE files DROP COLUMN IF EXISTS thumbnail_file_id`,
	}

	for _, sql := range statements {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

// 做一些GORM AutoMigrate 不方便做的SQL操作
func InitializeSchema(db *gorm.DB, embeddingDimensions int) error {
	statements := []string{
		// Full-text search vector column (generated, stored)
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'files' AND column_name = 'search_vector'
			) THEN
				ALTER TABLE files ADD COLUMN search_vector tsvector
					GENERATED ALWAYS AS (
						setweight(to_tsvector('simple', coalesce(file_name, '')), 'A') ||
						setweight(to_tsvector('simple', coalesce(caption, '')), 'B')
					) STORED;
			END IF;
		END $$`,

		// Full-text search GIN index
		`CREATE INDEX IF NOT EXISTS idx_files_search
		 ON files USING GIN(search_vector)`,
	}

	// pgvector extension and embedding column (only when vector search is enabled)
	if embeddingDimensions > 0 {
		statements = append(statements,
			// Enable the pgvector extension
			`CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA extensions`,

			// Embedding column using halfvec for half-precision storage
			fmt.Sprintf(`DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_name = 'files' AND column_name = 'embedding'
				) THEN
					ALTER TABLE files ADD COLUMN embedding extensions.halfvec(%d);
				END IF;
			END $$`, embeddingDimensions),
		)
	}

	for _, sql := range statements {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}
