package repository

import (
	"gorm.io/gorm"
)

// MaintenanceRepo 承担跨表的运维 / 修复类 SQL,通常由 owner 命令触发
// 这些操作往往不能简单地落入单表 repo,所以单独抽出来
type MaintenanceRepo struct {
	db *gorm.DB
}

// NewMaintenanceRepo 创建一个 MaintenanceRepo
func NewMaintenanceRepo(db *gorm.DB) *MaintenanceRepo {
	return &MaintenanceRepo{db: db}
}

// CapSyncResult 是 /cap_sync 命令的返回结构
type CapSyncResult struct {
	FilesUpdated    int64
	MessagesUpdated int64
}

// CaptionSync 为同属一个 media_group_id 的文件/消息回填 caption
//
// Telegram album 会拆成多条独立消息推给 bot,但通常只有第一条带 caption
// 导致其余文件入库后 caption 字段为空。该方法用一次性 SQL 完成回填:
//  1. 临时表 _cap_src 从 messages 收集 (media_group_id, caption)
//     同组多 caption 时取 received_at 最早 + id 最小的一条(对应"第一条")
//  2. 用临时表回填 files.caption(只更新 caption 为空的行)
//  3. 同样回填 messages.caption(只更新 caption 为空的行)
//
// 整体放在一个事务里,确保临时表对后续 UPDATE 可见
// 跳过 media_group_id 为 NULL 或空串的记录(那些是非 album 消息,无组可同步)
func (r *MaintenanceRepo) CaptionSync() (CapSyncResult, error) {
	var result CapSyncResult
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 临时表(ON COMMIT DROP 配合 tx 结束自动清理)
		if err := tx.Exec(`
			CREATE TEMP TABLE _cap_src ON COMMIT DROP AS
			SELECT DISTINCT ON (m.media_group_id) m.media_group_id, m.caption
			FROM messages m
			WHERE m.media_group_id IS NOT NULL
			  AND m.media_group_id <> ''
			  AND m.caption IS NOT NULL
			  AND m.caption <> ''
			ORDER BY m.media_group_id, m.received_at ASC, m.id ASC
		`).Error; err != nil {
			return err
		}

		// 回填 files
		res := tx.Exec(`
			UPDATE files f
			SET caption = src.caption
			FROM messages m, _cap_src src
			WHERE f.id = m.file_id
			  AND m.media_group_id = src.media_group_id
			  AND (f.caption IS NULL OR f.caption = '')
		`)
		if res.Error != nil {
			return res.Error
		}
		result.FilesUpdated = res.RowsAffected

		// 回填 messages 同组其他空 caption 行
		res = tx.Exec(`
			UPDATE messages m
			SET caption = src.caption
			FROM _cap_src src
			WHERE m.media_group_id = src.media_group_id
			  AND (m.caption IS NULL OR m.caption = '')
		`)
		if res.Error != nil {
			return res.Error
		}
		result.MessagesUpdated = res.RowsAffected

		return nil
	})
	return result, err
}
