package repository

import "tg-drive-bot/internal/model"

// This file declares the repository interfaces consumed by the service layer, plus the
// UnitOfWork abstraction used to run multi-repo work inside a single DB transaction.
//
// Every concrete implementation (*UserRepo, *FileRepo, etc.) must satisfy the matching
// interface. Service code should depend on the interfaces only — never on the concrete
// struct types — so that (a) unit tests can substitute fakes, and (b) a future swap of the
// ORM / storage engine only touches the repository package.

// UserRepository covers CRUD + whitelist queries for model.User.
type UserRepository interface {
	Create(user *model.User) error
	GetByTelegramID(telegramID int64) (*model.User, error)
	GetByID(id uint) (*model.User, error)
	Update(user *model.User) error
	Delete(id uint) error
	DeleteByTelegramID(telegramID int64) error
	ListAll(page, pageSize int) ([]model.User, int64, error)
	CountFilesByUserID(userID uint) (int64, error)
}

// FileRepository covers CRUD + listing + search + stats for model.File.
type FileRepository interface {
	Create(file *model.File) error
	GetByID(id uint) (*model.File, error)
	GetByUniqueIDAndUser(fileUniqueID string, userID uint) (*model.File, error)
	DeleteByIDAndUser(id, userID uint) (rowsAffected int64, err error)
	ExistsByUserAndFileUniqueID(userID uint, fileUniqueID string) (bool, error)
	ListByUser(userID uint, fileType string, page, pageSize int) ([]model.File, int64, error)
	RandomByUserAndTypes(userID uint, fileTypes []string, limit int) ([]model.File, error)
	Search(userID uint, query, fileType string, page, pageSize int) ([]model.File, int64, error)
	SearchFallback(userID uint, query, fileType string, page, pageSize int) ([]model.File, int64, error)
	VectorSearch(userID uint, queryVec []float64, fileType string, threshold float64, page, pageSize int) ([]model.File, int64, error)
	GetStatsByUser(userID uint) ([]FileTypeStats, error)
	GetTotalStatsByUser(userID uint) (*TotalStats, error)
	// 全库扫:用于 owner 命令 /emb_re 和 /emb_sync 的批量 embedding 任务
	ListIDsMissingEmbedding() ([]uint, error)
	ListAllFileIDs() ([]uint, error)
}

// FileCopyRepository covers the file_copies multi-copy table.
type FileCopyRepository interface {
	CreateBatch(copies []model.FileCopy) error
	ListByFile(fileID uint) ([]model.FileCopy, error)
	DeleteByFileID(fileID uint) error
}

// MessageRepository covers the messages observational-metadata table.
type MessageRepository interface {
	Create(msg *model.Message) error
	DeleteByFileID(fileID uint) error
}

// MaintenanceRepository 承载 owner 级别的跨表运维操作
// 这些操作往往涉及多张表的批量 SQL,不适合归到任何单表 repo
type MaintenanceRepository interface {
	CaptionSync() (CapSyncResult, error)
}

// Repos aggregates every repository interface. An instance represents a single
// consistency scope: either the root (db-backed, each call runs in its own short tx)
// or a tx-bound snapshot obtained via UnitOfWork.WithTx.
//
// Service code should accept *Repos or a subset rather than individual repos, so that
// mixed work can move in and out of transactions without changing call sites.
type Repos struct {
	User        UserRepository
	File        FileRepository
	FileCopy    FileCopyRepository
	Message     MessageRepository
	Maintenance MaintenanceRepository
}

// UnitOfWork runs a function inside a database transaction, passing a tx-bound *Repos
// whose repository calls all execute against the same transaction.
//
// Semantics:
//   - The passed closure should return nil to commit, non-nil to roll back.
//   - The returned error is the closure's error (or a tx framework error).
//   - Nested WithTx calls reuse the outer transaction; they do not start a nested tx.
//     (This mirrors GORM's default behavior — callers needing true savepoints must
//     add them explicitly.)
type UnitOfWork interface {
	WithTx(fn func(repos *Repos) error) error
}
