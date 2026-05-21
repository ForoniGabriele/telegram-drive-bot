package repository

import (
	"encoding/json"
	"errors"
	"math/rand/v2"
	"time"

	"tg-drive-bot/internal/model"

	"gorm.io/gorm"
)

// randomSamplingThreshold 是一个行数（记录数）的阈值
// 在此阈值以下，RandomByUserAndTypes 会使用简单的 ORDER BY RANDOM()以提高性能
// 当数量超过此阈值时，RandomByUserAndTypes 会切换为基于主键的采样方式
const randomSamplingThreshold = 500

// FileRepo is the GORM-backed implementation of FileRepository.
type FileRepo struct {
	db *gorm.DB
}

// NewFileRepo creates a FileRepo bound to the given DB or transaction handle.
func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

// Create inserts a new file record.
func (r *FileRepo) Create(file *model.File) error {
	return r.db.Create(file).Error
}

// GetByID finds a file by internal ID. Returns nil, nil if not found.
func (r *FileRepo) GetByID(id uint) (*model.File, error) {
	var file model.File
	err := r.db.First(&file, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// GetByIDAndUser 按主键查找文件,但要求文件归属 userID
// 用于来自 callback_data 的 fileID 取文件 -- callback_data 可被客户端伪造,
// 不能信任,必须在 SQL 层用 user_id 限定避免越权读取他人文件
// 不存在或不属于 userID 时统一返回 (nil, nil)
func (r *FileRepo) GetByIDAndUser(id, userID uint) (*model.File, error) {
	var file model.File
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// GetByUniqueIDAndUser Telegram 的 file_unique_id 和 user id查找文件
func (r *FileRepo) GetByUniqueIDAndUser(fileUniqueID string, userID uint) (*model.File, error) {
	var file model.File
	err := r.db.
		Where("file_unique_id = ? AND user_id = ?", fileUniqueID, userID).
		First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// DeleteByIDAndUser deletes a file row scoped by (id, user_id) and returns the rows affected.
// Callers use rowsAffected == 0 to detect "not found or not owned" and respond without
// leaking which of the two it was.
func (r *FileRepo) DeleteByIDAndUser(id, userID uint) (int64, error) {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.File{})
	return res.RowsAffected, res.Error
}

// ExistsByUserAndFileUniqueID checks if a file with the given unique_id already exists for the user.
func (r *FileRepo) ExistsByUserAndFileUniqueID(userID uint, fileUniqueID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.File{}).
		Where("user_id = ? AND file_unique_id = ?", userID, fileUniqueID).
		Count(&count).Error
	return count > 0, err
}

// ListByUser returns files for a user with optional type filter, paginated.
func (r *FileRepo) ListByUser(userID uint, fileType string, page, pageSize int) ([]model.File, int64, error) {
	db := r.db.Where("user_id = ?", userID)

	if fileType != "" && fileType != "all" {
		db = db.Where("file_type = ?", fileType)
	}

	var total int64
	if err := db.Model(&model.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []model.File
	err := db.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&files).Error

	return files, total, err
}

// 返回最多不超过 `limit` 个由 userID 拥有且 file_type 在 fileTypes 中的随机文件
// 如果符合条件的文件池大小小于 limit，则返回更少的数据行（或空切片）
//
// 算法：对于小型文件池（数量 <= randomSamplingThreshold），它会降级使用
// ORDER BY RANDOM() LIMIT n 以提高查询速度
// 对于较大的文件池，它会在 [min_id, max_id] 区间内对主键进行随机抽样，并通过
// "WHERE id >= ? ORDER BY id LIMIT 1" 将每个候选值映射到最邻近的实际数据行，
// 这样可以兼容由于删除操作导致的 ID 不连续（空洞）
// 这种抽样机制避免了在大表上使用 ORDER BY RANDOM() 所引发的全局全表扫描，
// 从而确保随机检索效率不会随着数据量的增长而退化
func (r *FileRepo) RandomByUserAndTypes(userID uint, fileTypes []string, limit int) ([]model.File, error) {
	if limit <= 0 || len(fileTypes) == 0 {
		return nil, nil
	}

	base := r.db.Model(&model.File{}).
		Where("user_id = ? AND file_type IN ?", userID, fileTypes)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, nil
	}

	if total <= randomSamplingThreshold {
		var files []model.File
		err := base.Order("RANDOM()").Limit(limit).Find(&files).Error
		return files, err
	}

	var bounds struct {
		MinID uint
		MaxID uint
	}
	if err := base.Select("MIN(id) AS min_id, MAX(id) AS max_id").Scan(&bounds).Error; err != nil {
		return nil, err
	}
	if bounds.MaxID == 0 || bounds.MinID == 0 {
		return nil, nil
	}

	// 对候选值进行过采样（超额抽样），以吸收由于删除操作产生的 ID 空洞，
	// 以及最邻近行解析带来的重复碰撞。即使频繁的删除操作导致 ID 范围变得
	// 非常稀疏，采用 3 倍数量加上最低 5 个的保底（limit*3 + 5）就足以确保
	// 可靠地填满所需的 limit 个文件
	candidateCount := limit*3 + 5
	span := bounds.MaxID - bounds.MinID + 1
	seen := make(map[uint]struct{}, limit)
	results := make([]model.File, 0, limit)

	for attempt := 0; attempt < candidateCount && len(results) < limit; attempt++ {
		offset := uint(rand.Uint64N(uint64(span)))
		candidate := bounds.MinID + offset

		var file model.File
		err := r.db.
			Where("user_id = ? AND file_type IN ? AND id >= ?", userID, fileTypes, candidate).
			Order("id ASC").
			Limit(1).
			Find(&file).Error
		if err != nil {
			return nil, err
		}
		if file.ID == 0 {
			continue
		}
		if _, dup := seen[file.ID]; dup {
			continue
		}
		seen[file.ID] = struct{}{}
		results = append(results, file)
	}

	return results, nil
}

// Search performs full-text search on file_name and caption using PostgreSQL tsvector.
func (r *FileRepo) Search(userID uint, query string, fileType string, page, pageSize int) ([]model.File, int64, error) {
	db := r.db.Where("user_id = ?", userID).
		Where("search_vector @@ plainto_tsquery('simple', ?)", query)

	if fileType != "" && fileType != "all" {
		db = db.Where("file_type = ?", fileType)
	}

	var total int64
	if err := db.Model(&model.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []model.File
	err := db.
		Order(gorm.Expr("ts_rank(search_vector, plainto_tsquery('simple', ?)) DESC", query)).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&files).Error

	return files, total, err
}

// SearchFallback performs ILIKE-based fuzzy search as a fallback when FTS is unavailable.
func (r *FileRepo) SearchFallback(userID uint, query string, fileType string, page, pageSize int) ([]model.File, int64, error) {
	pattern := "%" + query + "%"
	db := r.db.Where("user_id = ?", userID).
		Where("(file_name ILIKE ? OR caption ILIKE ?)", pattern, pattern)

	if fileType != "" && fileType != "all" {
		db = db.Where("file_type = ?", fileType)
	}

	var total int64
	if err := db.Model(&model.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []model.File
	err := db.Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&files).Error

	return files, total, err
}

// FileTypeStats holds the count and total size for a file type.
type FileTypeStats struct {
	FileType  string
	Count     int64
	TotalSize int64
}

// GetStatsByUser returns per-type file statistics for a user.
func (r *FileRepo) GetStatsByUser(userID uint) ([]FileTypeStats, error) {
	var stats []FileTypeStats
	err := r.db.Model(&model.File{}).
		Select("file_type, COUNT(*) as count, COALESCE(SUM(file_size), 0) as total_size").
		Where("user_id = ?", userID).
		Group("file_type").
		Find(&stats).Error
	return stats, err
}

// TotalStats holds aggregate stats (total count, total size, earliest, latest).
// Earliest/Latest 为 nil 表示该用户没有任何文件
type TotalStats struct {
	TotalCount int64
	TotalSize  int64
	Earliest   *time.Time
	Latest     *time.Time
}

// GetTotalStatsByUser returns aggregate stats for a user.
func (r *FileRepo) GetTotalStatsByUser(userID uint) (*TotalStats, error) {
	var stats TotalStats
	err := r.db.Model(&model.File{}).
		Select("COUNT(*) as total_count, COALESCE(SUM(file_size), 0) as total_size, MIN(created_at) as earliest, MAX(created_at) as latest").
		Where("user_id = ?", userID).
		Scan(&stats).Error
	return &stats, err
}

// VectorSearch 使用 pgvector 的余弦距离运算符 (<=>) 执行语义搜索
// 仅考虑 embedding 非空 (non-NULL) 的文件。结果按余弦距离升序排序
// (即最相似的排在最前面)。如果 threshold > 0,则排除距离大于该阈值的结果
func (r *FileRepo) VectorSearch(userID uint, queryVec []float64, fileType string, threshold float64, page, pageSize int) ([]model.File, int64, error) {
	vecJSON, err := json.Marshal(queryVec)
	if err != nil {
		return nil, 0, err
	}
	vecStr := string(vecJSON)

	db := r.db.Where("user_id = ? AND embedding IS NOT NULL", userID)

	if fileType != "" && fileType != "all" {
		db = db.Where("file_type = ?", fileType)
	}

	if threshold > 0 {
		db = db.Where("embedding <=> ?::extensions.halfvec <= ?", vecStr, threshold)
	}

	var total int64
	if err := db.Model(&model.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var files []model.File
	err = db.
		Order(gorm.Expr("embedding <=> ?::extensions.halfvec", vecStr)).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&files).Error

	return files, total, err
}

// ListIDsMissingEmbedding 返回 embedding 列为 NULL 的全部 file id
// 用于 owner 命令 /emb_sync 的"补缺失"路径
func (r *FileRepo) ListIDsMissingEmbedding() ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.File{}).
		Where("embedding IS NULL").
		Order("id").
		Pluck("id", &ids).Error
	return ids, err
}

// ListAllFileIDs 返回全部 file id(不分用户)
// 用于 owner 命令 /emb_re 的"全量重做"路径
func (r *FileRepo) ListAllFileIDs() ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.File{}).
		Order("id").
		Pluck("id", &ids).Error
	return ids, err
}

// EmbeddingSource 是生成 embedding 所需的文本字段子集
// 不包含数据库主键以外的字段, 调用方按 fileID 配对结果
type EmbeddingSource struct {
	FileName string
	Title    string
	Caption  string
}

// GetEmbeddingSource 读取生成 embedding 所需的文本字段
// 找不到时返回 (nil, nil), 调用方据此跳过该 fileID
func (r *FileRepo) GetEmbeddingSource(fileID uint) (*EmbeddingSource, error) {
	var src EmbeddingSource
	err := r.db.Table("files").
		Select("file_name, title, caption").
		Where("id = ?", fileID).
		Scan(&src).Error
	if err != nil {
		return nil, err
	}
	return &src, nil
}

// SaveEmbedding 将一个 fileID 的 embedding 向量写回 files 表
// vecJSON 必须是 json.Marshal 得到的 float64 数组字面量(例如 "[0.1,0.2,...]"),
// 由 service 层产出 -- 这里只负责强类型转换为 pgvector 的 halfvec
func (r *FileRepo) SaveEmbedding(fileID uint, vecJSON string) error {
	return r.db.Exec(
		"UPDATE files SET embedding = ?::extensions.halfvec WHERE id = ?",
		vecJSON, fileID,
	).Error
}
