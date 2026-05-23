package msg

var Zh = Catalog{
	// Generic errors
	ErrUnknown:     "❌ 未知错误",
	ErrInvalidArgs: "❌ 参数错误",
	ErrInvalidID:   "❌ 参数错误, user_id 必须是数字",
	ErrOperation:   "❌ 操作失败",

	// File-related
	FileNotFound:         "❌ 文件不存在",
	FileNotFoundOrNoPerm: "❌ 文件不存在或无权限",
	FileSendFailed:       "❌ 文件发送失败, 可能已从存储中删除",
	FileUnavailable:      "❌ 文件暂时不可用, 所有存储位置都无法访问",
	FileSaveFailed:       "❌ 文件保存失败, 请稍后重试",
	FileAlreadyExists:    "⚠️ 文件已存在, 无需重复保存",
	FileDeleteFailed:     "❌ 删除失败, 请稍后重试",
	FileDeleted:          "✅ 文件已删除",
	ListFetchFailed:      "❌ 获取文件列表失败",
	SearchFailed:         "❌ 搜索失败",
	SearchEmptyQuery:     "❌ 请输入搜索关键词, 例如: /search 财务报告",
	SearchExpired:        "⏰ 搜索已过期, 请重新搜索",
	StatsFetchFailed:     "❌ 获取统计信息失败",

	// Random
	RandInvalidNumber: "❌ 数量参数错误, 用法: %[1]s [数量], 默认 5, 上限 %[2]d",
	RandNoMedia:       "❌ 没有找到任何符合条件的媒体",
	RandPartial:       "⚠️ 仅找到 %[1]d 条符合条件的媒体",
	RandFetchFailed:   "❌ 随机获取失败",

	// User management
	AddUserUsage:         "❌ 参数错误, 用法: /adduser <user_id>",
	RemoveUserUsage:      "❌ 参数错误, 用法: /removeuser <user_id>",
	AddUserFailed:        "❌ 添加用户失败",
	RemoveUserFailed:     "❌ 移除用户失败",
	DeleteUserFailed:     "❌ 删除用户失败",
	PromoteFailed:        "❌ 提升失败",
	DemoteFailed:         "❌ 降级失败",
	UserListFetchFailed:  "❌ 获取用户列表失败",
	UserNotFound:         "❌ 用户不存在",
	CannotRemoveOwner:    "❌ 无法移除 Owner",
	CannotModifyOwner:    "❌ 无法修改 Owner",
	OnlyOwnerRemoveAdmin: "❌ 只有 Owner 才能移除管理员",
	OnlyOwnerDemoteAdmin: "❌ 只有 Owner 才能降级管理员",
	AlreadyAdmin:         "⚠️ 用户已是管理员",
	AlreadyUser:          "⚠️ 用户已是普通用户",
	Promoted:             "✅ 已提升为管理员",
	Demoted:              "✅ 已降级为普通用户",

	UserAlreadyWhitelisted: "⚠️ 用户 %[1]d 已在白名单中",
	UserAdded:              "✅ 已添加用户 %[1]d 到白名单",
	UserNotInWhitelist:     "❌ 用户 %[1]d 不在白名单中",
	UserRemoved:            "✅ 已将用户 %[1]d 从白名单移除",

	// Telegra.ph
	TphUsage:        "❌ 请提供 telegra.ph 链接, 用法: /tph <url>",
	TphInvalidURL:   "❌ 无效的链接, 仅支持 telegra.ph 域名",
	TphFetchFailed:  "❌ 获取文章失败, 请检查链接或稍后重试",
	TphEmptyContent: "❌ 文章内容为空",
	TphFetching:     "⏳ 正在抓取文章...",
	TphSaved:        "✅ 已保存: %[1]s",

	// Owner maintenance
	EmbBatchStartedRe:       "🚀 已启动 /emb_re(全量重做 embedding), 扫描中...",
	EmbBatchStartedSync:     "🚀 已启动 /emb_sync(补缺失 embedding), 扫描中...",
	EmbBatchBusy:            "⚠️ 已有 embedding 批量任务在运行, 请稍后再试",
	EmbBatchProgress:        "⏳ 进度 %[1]d/%[2]d (%[3]s)",
	EmbBatchEmpty:           "ℹ️ 没有需要处理的文件",
	EmbBatchDone:            "✅ embedding 批量完成: 共处理 %[1]d 个文件",
	EmbBatchFailed:          "❌ embedding 批量失败: %[1]v",
	EmbVectorSearchDisabled: "❌ 向量搜索未启用, 无法运行 embedding 批量",

	CapSyncStarted: "🚀 开始 /cap_sync(补全媒体组 caption)...",
	CapSyncBusy:    "⚠️ 已有 cap_sync 任务在运行, 请稍后再试",
	CapSyncDone:    "✅ caption 回填完成: files 更新 %[1]d 行, messages 更新 %[2]d 行\n💡 caption 已变, 可考虑跑 /emb_sync 让新内容进入向量索引",
	CapSyncFailed:  "❌ cap_sync 失败: %[1]v",

	Welcome: `👋 欢迎使用 TG Drive Bot!

你可以直接发送文件/图片/视频等媒体, 我会自动帮你存储索引.

📌 可用命令:
/list - 浏览我的文件库
/search <关键词> - 搜索文件
/stats - 查看统计信息
/rand [数量] - 随机返回媒体(默认5, 上限10)
/randv [数量] - 随机返回视频
/randp [数量] - 随机返回图片
/tph <url> - 抓取 telegra.ph 文章为 Markdown 并保存
/lang [zh|en] - 查看或切换语言`,

	WelcomeAdminExtra: `

🛡️ 管理员命令:
/adduser <user_id> - 添加白名单用户
/removeuser <user_id> - 移除白名单用户
/listuser - 查看用户列表`,

	WelcomeOwnerExtra: `

👑 Owner 命令:
/emb_re - 全量重做 embedding
/emb_sync - 补缺失 embedding
/cap_sync - 补全媒体组 caption`,

	// B-class templates lifted from handlers (all using %[N] indexed verbs)
	// media.go
	SavedWithName: "✅ 已保存: %[1]s (%[2]s %[3]s, %[4]s)", // name, icon, typeName, size
	SavedTypeOnly: "✅ 已保存: %[1]s %[2]s (%[3]s)",        // icon, typeName, size

	// list.go
	ListEmpty:         "📁 我的文件库\n\n还没有存储任何文件, 直接发送文件给我即可开始!",
	ListEmptyType:     "📁 我的文件库\n\n没有 %[1]s %[2]s 类型的文件", // icon, typeName
	ListHeader:        "📁 我的文件库 (共 %[1]d 个文件)\n\n",       // total
	FileCaptionHeader: "📋 文件信息\n━━━━━━━━━━━━━━━\n%[1]s 文件名: %[2]s\n📦 大小: %[3]s\n📅 存入时间: %[4]s\n━━━━━━━━━━━━━━━",

	// search.go
	SearchEmpty:  "🔍 搜索 \"%[1]s\" 的结果\n\n未找到匹配的文件",     // query
	SearchHeader: "🔍 搜索 \"%[1]s\" 的结果 (共 %[2]d 个)\n\n", // query, total

	StatsBlock: "📊 你的文件库统计\n\n", // header only; remaining lines composed inline

	// admin_user.go
	UserListHeader:    "👥 用户管理 (共 %[1]d 人)\n\n",
	UserListEmpty:     "👥 用户管理\n\n暂无用户",
	UserInfoCard:      "👤 用户信息\n━━━━━━━━━━━━━━━\n📛 用户名: %[1]s %[2]s\n🆔 Telegram ID: %[3]d\n🔑 角色: %[4]s\n📦 文件数: %[5]d\n📅 加入时间: %[6]s",
	BtnBackToUserList: "◀ 返回用户列表",
	BtnDeleteUser:     "🗑 删除用户",
	BtnPromoteToAdmin: "⬆️ 提升为管理员",
	BtnDemoteToUser:   "⬇️ 降级为普通用户",

	// ui/keyboard.go
	BtnDelete:        "🗑 删除",
	BtnConfirmDelete: "✅ 确认删除",
	BtnCancel:        "❌ 取消",

	// ui/paginator.go
	BtnFilterDocument: "📄 文档",
	BtnFilterAudio:    "🎵 音频",
	BtnFilterVideo:    "🎬 视频",
	BtnFilterPhoto:    "🖼 图片",
	BtnFilterAll:      "全部",
	BtnPagePrev:       "◀ 上一页",
	BtnPageNext:       "下一页 ▶",

	// util/format.go
	FileTypeDocument:  "文档",
	FileTypeAudio:     "音频",
	FileTypeVideo:     "视频",
	FileTypePhoto:     "图片",
	FileTypeAnimation: "动图",
	FileTypeVoice:     "语音",
	FileTypeVideoNote: "视频笔记",
	FileTypeUnknown:   "未知",

	// user_service.go
	RoleAdminName: "管理员",
	RoleUserName:  "用户",

	// /lang
	LangUsage:   "用法: /lang [zh|en]\n查看当前语言或切换到指定语言.",
	LangCurrent: "🌐 当前语言: %[1]s",
	LangSet:     "✅ 已切换语言到: %[1]s",
	LangInvalid: "❌ 不支持的语言, 仅支持 zh / en",
}
