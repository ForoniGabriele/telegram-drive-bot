package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"tg-drive-bot/internal/model"
	"tg-drive-bot/internal/repository"
	"tg-drive-bot/internal/storage"

	tele "gopkg.in/telebot.v4"
)

// ErrFileNotFound is returned when a file lookup fails ownership/existence checks.
var ErrFileNotFound = errors.New("file not found or not owned by user")

// FileInfo is a DTO for transferring file metadata between handler and service layers.
// This avoids coupling the service layer to telebot types.
type FileInfo struct {
	FileType     string
	BotFileID    string // Telegram 的 file_id(长 base64-like 字符串)
	FileUniqueID string
	FileName     string
	MimeType     string
	FileSize     int64
	Duration     int
	Width        int
	Height       int
	Performer    string
	Title        string
	Caption      string
}

// MessageInfo is a DTO for the message context of a file submission.
type MessageInfo struct {
	MessageID    int
	ChatID       int64
	FromUserID   int64
	FromUsername string
	Caption      string
	MediaGroupID string
	ForwardFrom  string
}

// asyncConcurrency 限制了可以同时进行的异步频道操作（转发/删除）的数量
// 这可以防止用户在短时间内突发上传/删除时产生goroutine爆炸，
// 避免因此压垮 Telegram API 并触发 429 错误
const asyncConcurrency = 5

// FileService 处理文件业务逻辑
//
// 它持有对持久层的两个引用：
//
//   - repos 是根作用域、自动提交的 Repos，用于单条语句的读取以及
//     事务提交后的best-effort写入
//   - uow 用于将多步骤的写入作为一个单一的原子单元运行（事务）
//
// 该服务从不直接接触 *gorm.DB —— 所有的事务作用域控制都通过
// uow.WithTx 进行，从而使持久层实现保持可替换性
type FileService struct {
	repos        *repository.Repos
	uow          repository.UnitOfWork
	embedding    *EmbeddingService // nil when vector search is disabled
	asyncForward bool              // forward to storage channels in background goroutine
	asyncDelete  bool              // delete from storage channels in background goroutine
	asyncSem     chan struct{}     // limits concurrent async channel operations; nil when both flags are false
}

// 创建一个新的 FileService
// 当禁用向量搜索时，embedding 可能为 nil
// asyncForward / asyncDelete 用于控制频道存储的 I/O 操作是否在goroutine中运行
func NewFileService(repos *repository.Repos, uow repository.UnitOfWork, embedding *EmbeddingService, asyncForward, asyncDelete bool) *FileService {
	var sem chan struct{}
	if asyncForward || asyncDelete {
		sem = make(chan struct{}, asyncConcurrency)
	}
	return &FileService{
		repos:        repos,
		uow:          uow,
		embedding:    embedding,
		asyncForward: asyncForward,
		asyncDelete:  asyncDelete,
		asyncSem:     sem,
	}
}

// SaveFileResult holds the result of a SaveFile operation.
type SaveFileResult struct {
	Status   string // "saved", "duplicate"
	File     *model.File
	FileName string
}

// SaveFile 检查重复项，然后保存文件记录、存储副本和消息上下文
//
// 在同步模式下（默认）：
//   - 先转发到存储频道；如果完全失败则返回错误（数据库中不会写入任何内容）。
//   - 在单个事务中写入 files + file_copies 记录。
//
// 在异步转发模式下（ASYNC_FORWARD=true，仅适用于频道存储）：
//   - 立即写入 files 记录（不写副本）
//   - 启动后台协程（goroutine）转发到频道并写入 file_copies 记录
//   - 用户立即收到成功响应；频道副本的保存是best-effort的
//
// 在直接模式下，store.ForwardToStorage 是一个空操作（no-op）并返回空副本，
// 因此两条路径最终都会合流为“仅写入 files 记录”。
//
// 消息上下文行总是在主事务提交之后写入（也是best-effort）
func (s *FileService) SaveFile(userID uint, info *FileInfo, msgInfo *MessageInfo, bot tele.API, originalMsg *tele.Message, store storage.Storage) (*SaveFileResult, error) {
	exists, err := s.repos.File.ExistsByUserAndFileUniqueID(userID, info.FileUniqueID)
	if err != nil {
		return nil, err
	}
	if exists {
		return &SaveFileResult{Status: "duplicate"}, nil
	}

	file := &model.File{
		UserID:       userID,
		FileUniqueID: info.FileUniqueID,
		BotFileID:    info.BotFileID,
		FileType:     info.FileType,
		FileName:     info.FileName,
		MimeType:     info.MimeType,
		FileSize:     info.FileSize,
		Duration:     info.Duration,
		Width:        info.Width,
		Height:       info.Height,
		Performer:    info.Performer,
		Title:        info.Title,
		Caption:      info.Caption,
	}

	if s.asyncForward {
		// Async path: write file record only, forward to channels in background.
		txErr := s.uow.WithTx(func(r *repository.Repos) error {
			return r.File.Create(file)
		})
		if txErr != nil {
			return nil, txErr
		}
		go s.asyncForwardCopies(file.ID, bot, originalMsg, store)
	} else {
		// Sync path: forward first, then write file + copies together.
		copies, fwdErr := store.ForwardToStorage(bot, originalMsg)
		if fwdErr != nil {
			return nil, fwdErr
		}

		txErr := s.uow.WithTx(func(r *repository.Repos) error {
			if err := r.File.Create(file); err != nil {
				return err
			}
			if len(copies) > 0 {
				rows := make([]model.FileCopy, 0, len(copies))
				for _, c := range copies {
					rows = append(rows, model.FileCopy{
						FileID:        file.ID,
						StorageChatID: c.ChatID,
						StorageMsgID:  c.MsgID,
					})
				}
				if err := r.FileCopy.CreateBatch(rows); err != nil {
					return fmt.Errorf("create file_copies: %w", err)
				}
			}
			return nil
		})
		if txErr != nil {
			return nil, txErr
		}
	}

	msg := &model.Message{
		FileID:       file.ID,
		MessageID:    msgInfo.MessageID,
		ChatID:       msgInfo.ChatID,
		FromUserID:   msgInfo.FromUserID,
		FromUsername: msgInfo.FromUsername,
		Caption:      msgInfo.Caption,
		MediaGroupID: msgInfo.MediaGroupID,
		ForwardFrom:  msgInfo.ForwardFrom,
		ReceivedAt:   time.Now(),
	}
	if err := s.repos.Message.Create(msg); err != nil {
		slog.Warn("file saved but message record failed", "error", err, "file_db_id", file.ID)
	}

	result := &SaveFileResult{
		Status:   "saved",
		File:     file,
		FileName: info.FileName,
	}

	// Enqueue async embedding generation (non-blocking, fire-and-forget).
	if s.embedding != nil {
		s.embedding.Enqueue(file.ID)
	}

	return result, nil
}

// CheckDuplicate checks if the file already exists for a user.
func (s *FileService) CheckDuplicate(userID uint, fileUniqueID string) (bool, error) {
	return s.repos.File.ExistsByUserAndFileUniqueID(userID, fileUniqueID)
}

// GetFilesByUser returns paginated files for a user with optional type filter.
func (s *FileService) GetFilesByUser(userID uint, fileType string, page, pageSize int) ([]model.File, int64, error) {
	return s.repos.File.ListByUser(userID, fileType, page, pageSize)
}

// 返回最多不超过 `limit` 个符合任何给定文件类型的随机文件
// 当用户在所请求的类型中没有任何文件时，返回一个空切片（并且 error 为 nil），
// 如果文件池的实际数量小于 limit，则可能会返回少于 `limit` 行的记录
func (s *FileService) GetRandomFiles(userID uint, fileTypes []string, limit int) ([]model.File, error) {
	return s.repos.File.RandomByUserAndTypes(userID, fileTypes, limit)
}

// SearchFiles 使用三级降级链执行搜索：
//  1. 向量搜索（当启用且 API 可达时）—— 语义匹配
//  2. FTS 全文检索 —— 带有 ts_rank 权重的关键词匹配
//  3. ILIKE 模糊搜索 —— 最终保底的子字符串匹配
func (s *FileService) SearchFiles(ctx context.Context, userID uint, query, fileType string, page, pageSize int) ([]model.File, int64, error) {
	// Tier 1: vector search
	if s.embedding != nil {
		slog.Info("search [Tier 1]: starting vector search", "query", query)
		queryVec, err := s.embedding.GenerateEmbedding(ctx, query)
		if err == nil {
			files, total, err := s.repos.File.VectorSearch(userID, queryVec, fileType, s.embedding.Threshold(), page, pageSize)
			if err == nil && total > 0 {
				slog.Info("search [Tier 1]: vector search successful", "total", total)
				return files, total, nil
			}
			if err != nil {
				slog.Warn("search [Tier 1]: vector search query failed, falling back to FTS", "error", err)
			} else {
				slog.Info("search [Tier 1]: 0 matches found, falling back to FTS")
			}
		} else {
			slog.Warn("search [Tier 1]: embedding API failed, falling back to FTS", "error", err, "query", query)
		}
	}

	// 降级到关键词匹配链 (Tier 2 FTS + Tier 3 ILIKE)
	return s.searchKeywordOnly(userID, query, fileType, page, pageSize)
}

// SearchFilesNoVector 跳过向量搜索,直接走 FTS + ILIKE 降级链
// 对应用户命令 /ss,适合明确希望关键词精确匹配、不需要语义模糊的场景
func (s *FileService) SearchFilesNoVector(_ context.Context, userID uint, query, fileType string, page, pageSize int) ([]model.File, int64, error) {
	return s.searchKeywordOnly(userID, query, fileType, page, pageSize)
}

// searchKeywordOnly 是 FTS + ILIKE 两层降级的共享实现
// SearchFiles 在向量层降级时调用,SearchFilesNoVector 直接调用
func (s *FileService) searchKeywordOnly(userID uint, query, fileType string, page, pageSize int) ([]model.File, int64, error) {
	// Tier 2: FTS full-text search
	slog.Info("search [Tier 2]: starting FTS search", "query", query)
	files, total, err := s.repos.File.Search(userID, query, fileType, page, pageSize)
	if err == nil && total > 0 {
		slog.Info("search [Tier 2]: FTS search successful", "total", total)
		return files, total, nil
	}

	if err != nil {
		slog.Warn("search [Tier 2]: FTS search failed, falling back to ILIKE", "error", err, "query", query)
	} else {
		slog.Info("search [Tier 2]: 0 matches found, falling back to ILIKE")
	}

	// Tier 3: ILIKE fuzzy search
	slog.Info("search [Tier 3]: starting ILIKE fuzzy search", "query", query)
	files, total, err = s.repos.File.SearchFallback(userID, query, fileType, page, pageSize)
	if err == nil {
		slog.Info("search [Tier 3]: ILIKE search finished", "total", total)
	} else {
		slog.Error("search [Tier 3]: ILIKE search failed", "error", err)
	}
	return files, total, err
}

// GetFileByID returns a file by its internal ID without an ownership check.
// 仅供服务内部 / 系统路径使用(例如批量任务、维护命令)
// 来自用户的 fileID(尤其是 callback_data 中的 fileID)必须改用 GetFileByIDAndUser,
// 否则白名单用户可枚举主键越权下载他人文件
func (s *FileService) GetFileByID(id uint) (*model.File, error) {
	return s.repos.File.GetByID(id)
}

// GetFileByIDAndUser 按主键取文件并校验归属
// 不存在或不属于该 userID 时统一返回 (nil, nil) -- 调用方据此回 "文件不存在",
// 避免泄露 "存在但不属于你" 的信息
func (s *FileService) GetFileByIDAndUser(id, userID uint) (*model.File, error) {
	return s.repos.File.GetByIDAndUser(id, userID)
}

// GetFileByUniqueIDAndUser returns a file by its Telegram file_unique_id, scoped to userID.
// Returns nil, nil when the file doesn't exist or isn't owned by that user.
func (s *FileService) GetFileByUniqueIDAndUser(fileUniqueID string, userID uint) (*model.File, error) {
	return s.repos.File.GetByUniqueIDAndUser(fileUniqueID, userID)
}

// 返回文件的有序存储副本列表
// 当文件没有记录任何副本时返回 nil（包括直接存储模式的记录，以及频道模式下
// 副本丢失的记录 —— 后者会因为无法获取副本而触发调用方的“直接发送”备用方案）。
func (s *FileService) GetFileCopies(file *model.File) ([]model.CopyLocation, error) {
	rows, err := s.repos.FileCopy.ListByFile(file.ID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	locs := make([]model.CopyLocation, 0, len(rows))
	for _, r := range rows {
		locs = append(locs, model.CopyLocation{ChatID: r.StorageChatID, MsgID: r.StorageMsgID})
	}
	return locs, nil
}

// GetStats returns per-type file statistics for a user.
func (s *FileService) GetStats(userID uint) ([]FileTypeStats, error) {
	return s.repos.File.GetStatsByUser(userID)
}

// GetTotalStats returns aggregate stats for a user.
func (s *FileService) GetTotalStats(userID uint) (*TotalStats, error) {
	return s.repos.File.GetTotalStatsByUser(userID)
}

// DeleteFile 删除文件的数据库记录，并在成功后best-effort地删除其存储频道中的副本
//
// 数据库层约束（不变性）：
//   - file_copies、messages 和 files 表的对应行在单个事务中被删除以保持数据一致性
//   - 事务内部通过 DeleteByIDAndUser 强制进行所有权验证，因此用户无法通过
//     猜测 ID 来删除其他用户的文件。
//
// 存储频道的清理操作仅在数据库事务提交之后运行。即使清理失败也仅会记录日志，
// 而不会向上传播给用户，因为此时数据库中的记录已被删除，让用户感知失败也无法进行重试
//
// 当文件不存在或不属于该 userID 时，返回 ErrFileNotFound。
// 在频道模式下，若 DELETE_CHANNEL_COPIES=false，则存储层删除调用是一个空操作（no-op）
// 在直接模式下，存储层删除调用同样也是一个空操作（no-op）
func (s *FileService) DeleteFile(userID uint, fileID uint, bot tele.API, store storage.Storage) error {
	// 预读 copies 文件到内存中, 不然删记录的事务结束后就找不到这些文件了
	file, err := s.repos.File.GetByID(fileID)
	if err != nil {
		return err
	}
	if file == nil || file.UserID != userID {
		return ErrFileNotFound
	}

	copies, err := s.GetFileCopies(file)
	if err != nil {
		return fmt.Errorf("load copies: %w", err)
	}

	txErr := s.uow.WithTx(func(r *repository.Repos) error {
		if err := r.FileCopy.DeleteByFileID(file.ID); err != nil {
			return err
		}
		if err := r.Message.DeleteByFileID(file.ID); err != nil {
			return err
		}
		rows, err := r.File.DeleteByIDAndUser(file.ID, userID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrFileNotFound
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}

	// DB is committed. Storage cleanup is best-effort.
	if s.asyncDelete {
		go s.asyncDeleteCopies(file.ID, bot, store, copies)
	} else {
		if err := store.DeleteFromStorage(bot, copies); err != nil {
			slog.Warn("storage cleanup failed (DB already deleted, copies may remain)",
				"error", err, "file_db_id", file.ID)
		}
	}
	slog.Info("file deleted", "file_db_id", file.ID, "user_id", userID, "copies", len(copies))
	return nil
}

// 在后台goroutine中运行，用于将文件异步转发到存储频道
// 并持久化生成的副本位置记录。发生错误时仅记录日志而不会向上传播
// 因为文件记录已保存在数据库中，可以通过直接的 file_id 降级获取。
func (s *FileService) asyncForwardCopies(fileID uint, bot tele.API, originalMsg *tele.Message, store storage.Storage) {
	s.asyncSem <- struct{}{}
	defer func() { <-s.asyncSem }()

	copies, err := store.ForwardToStorage(bot, originalMsg)
	if err != nil {
		slog.Warn("async forward to storage failed (file saved, copies missing)",
			"error", err, "file_db_id", fileID)
		return
	}
	if len(copies) == 0 {
		return
	}

	rows := make([]model.FileCopy, 0, len(copies))
	for _, c := range copies {
		rows = append(rows, model.FileCopy{
			FileID:        fileID,
			StorageChatID: c.ChatID,
			StorageMsgID:  c.MsgID,
		})
	}
	if err := s.repos.FileCopy.CreateBatch(rows); err != nil {
		slog.Warn("async save file_copies failed (file saved, copies missing)",
			"error", err, "file_db_id", fileID)
	} else {
		slog.Info("async forward completed", "file_db_id", fileID, "copies", len(copies))
	}
}

// 在后台goroutine中运行，用于删除频道副本消息
// 发生错误时仅记录日志而不会向上传播——因为数据库记录已经删除了
func (s *FileService) asyncDeleteCopies(fileID uint, bot tele.API, store storage.Storage, copies []model.CopyLocation) {
	s.asyncSem <- struct{}{}
	defer func() { <-s.asyncSem }()

	if err := store.DeleteFromStorage(bot, copies); err != nil {
		slog.Warn("async storage cleanup failed (DB already deleted, copies may remain)",
			"error", err, "file_db_id", fileID)
	} else {
		slog.Info("async storage cleanup completed", "file_db_id", fileID, "copies", len(copies))
	}
}

// Re-export repository types for use by handler layer.
type FileTypeStats = repository.FileTypeStats
type TotalStats = repository.TotalStats
