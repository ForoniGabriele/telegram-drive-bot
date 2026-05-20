// Package msg centralizes user-facing text so handlers don't embed strings inline.
package msg

// Generic errors.
const (
	ErrUnknown     = "❌ 未知错误"
	ErrInvalidArgs = "❌ 参数错误"
	ErrInvalidID   = "❌ 参数错误, user_id 必须是数字"
	ErrOperation   = "❌ 操作失败"
)

// File-related messages.
const (
	FileNotFound         = "❌ 文件不存在"
	FileNotFoundOrNoPerm = "❌ 文件不存在或无权限"
	FileSendFailed       = "❌ 文件发送失败, 可能已从存储中删除"
	FileUnavailable      = "❌ 文件暂时不可用, 所有存储位置都无法访问"
	FileSaveFailed       = "❌ 文件保存失败, 请稍后重试"
	FileAlreadyExists    = "⚠️ 文件已存在, 无需重复保存"
	FileDeleteFailed     = "❌ 删除失败, 请稍后重试"
	FileDeleted          = "✅ 文件已删除"
	ListFetchFailed      = "❌ 获取文件列表失败"
	SearchFailed         = "❌ 搜索失败"
	SearchEmptyQuery     = "❌ 请输入搜索关键词, 例如: /search 财务报告"
	SearchExpired        = "⏰ 搜索已过期, 请重新搜索"
	StatsFetchFailed     = "❌ 获取统计信息失败"
)

// Random media messages.
const (
	RandInvalidNumber = "❌ 数量参数错误, 用法: %s [数量], 默认 5, 上限 %d"
	RandNoMedia       = "❌ 没有找到任何符合条件的媒体"
	RandPartial       = "⚠️ 仅找到 %d 条符合条件的媒体"
	RandFetchFailed   = "❌ 随机获取失败"
)

// User management messages.
const (
	AddUserUsage         = "❌ 参数错误, 用法: /adduser <user_id>"
	RemoveUserUsage      = "❌ 参数错误, 用法: /removeuser <user_id>"
	AddUserFailed        = "❌ 添加用户失败"
	RemoveUserFailed     = "❌ 移除用户失败"
	DeleteUserFailed     = "❌ 删除用户失败"
	PromoteFailed        = "❌ 提升失败"
	DemoteFailed         = "❌ 降级失败"
	UserListFetchFailed  = "❌ 获取用户列表失败"
	UserNotFound         = "❌ 用户不存在"
	CannotRemoveOwner    = "❌ 无法移除 Owner"
	CannotModifyOwner    = "❌ 无法修改 Owner"
	OnlyOwnerRemoveAdmin = "❌ 只有 Owner 才能移除管理员"
	OnlyOwnerDemoteAdmin = "❌ 只有 Owner 才能降级管理员"
	AlreadyAdmin         = "⚠️ 用户已是管理员"
	AlreadyUser          = "⚠️ 用户已是普通用户"
	Promoted             = "✅ 已提升为管理员"
	Demoted              = "✅ 已降级为普通用户"
)

// Telegra.ph fetch messages.
const (
	TphUsage        = "❌ 请提供 telegra.ph 链接, 用法: /tph <url>"
	TphInvalidURL   = "❌ 无效的链接, 仅支持 telegra.ph 域名"
	TphFetchFailed  = "❌ 获取文章失败, 请检查链接或稍后重试"
	TphEmptyContent = "❌ 文章内容为空"
	TphFetching     = "⏳ 正在抓取文章..."
	TphSaved        = "✅ 已保存: %s"
)

// Owner-only maintenance commands (/emb_re, /emb_sync, /cap_sync).
const (
	EmbBatchStartedRe   = "🚀 已启动 /emb_re(全量重做 embedding), 扫描中..."
	EmbBatchStartedSync = "🚀 已启动 /emb_sync(补缺失 embedding), 扫描中..."
	EmbBatchBusy        = "⚠️ 已有 embedding 批量任务在运行, 请稍后再试"
	EmbBatchProgress    = "⏳ 进度 %d/%d (%s)"
	EmbBatchEmpty       = "ℹ️ 没有需要处理的文件"
	EmbBatchDone        = "✅ embedding 批量完成: 共处理 %d 个文件"
	EmbBatchFailed      = "❌ embedding 批量失败: %v"

	CapSyncStarted    = "🚀 开始 /cap_sync(补全媒体组 caption)..."
	CapSyncBusy       = "⚠️ 已有 cap_sync 任务在运行, 请稍后再试"
	CapSyncDone       = "✅ caption 回填完成: files 更新 %d 行, messages 更新 %d 行\n💡 caption 已变, 可考虑跑 /emb_sync 让新内容进入向量索引"
	CapSyncFailed     = "❌ cap_sync 失败: %v"
)

// Start banner.
const Welcome = `👋 欢迎使用 TG Drive Bot!

你可以直接发送文件/图片/视频等媒体, 我会自动帮你存储索引.

📌 可用命令:
/list - 浏览我的文件库
/search <关键词> - 搜索文件
/stats - 查看统计信息
/rand [数量] - 随机返回媒体(默认5, 上限10)
/randv [数量] - 随机返回视频
/randp [数量] - 随机返回图片
/tph <url> - 抓取 telegra.ph 文章为 Markdown 并保存`

// WelcomeAdminExtra lists admin-only commands appended to Welcome for admins/owners.
const WelcomeAdminExtra = `

🛡️ 管理员命令:
/adduser <user_id> - 添加白名单用户
/removeuser <user_id> - 移除白名单用户
/listuser - 查看用户列表`

// WelcomeOwnerExtra lists owner-only commands appended after WelcomeAdminExtra for the owner.
const WelcomeOwnerExtra = `

👑 Owner 命令:
/emb_re - 全量重做 embedding
/emb_sync - 补缺失 embedding
/cap_sync - 补全媒体组 caption`
