package repository

import (
	"tg-drive-bot/internal/model"

	"gorm.io/gorm"
)

// FileCopyRepo is the GORM-backed implementation of FileCopyRepository.
type FileCopyRepo struct {
	db *gorm.DB
}

// NewFileCopyRepo creates a FileCopyRepo bound to the given DB or transaction handle.
func NewFileCopyRepo(db *gorm.DB) *FileCopyRepo {
	return &FileCopyRepo{db: db}
}

// CreateBatch inserts multiple file copy records in a single statement.
func (r *FileCopyRepo) CreateBatch(copies []model.FileCopy) error {
	if len(copies) == 0 {
		return nil
	}
	return r.db.Create(&copies).Error
}

// ListByFile returns all copies of a file, ordered by created_at ASC (insertion order).
func (r *FileCopyRepo) ListByFile(fileID uint) ([]model.FileCopy, error) {
	var copies []model.FileCopy
	err := r.db.Where("file_id = ?", fileID).
		Order("created_at ASC, id ASC").
		Find(&copies).Error
	return copies, err
}

// DeleteByFileID removes every copy belonging to the given file.
func (r *FileCopyRepo) DeleteByFileID(fileID uint) error {
	return r.db.Where("file_id = ?", fileID).Delete(&model.FileCopy{}).Error
}
