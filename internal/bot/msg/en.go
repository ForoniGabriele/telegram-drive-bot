package msg

// En is the English catalog. Field semantics mirror Zh; only the localized
// values differ. Format strings use indexed placeholders so we can freely
// reorder arguments per locale.
var En = Catalog{
	// Generic errors
	ErrUnknown:     "❌ Unknown error",
	ErrInvalidArgs: "❌ Invalid arguments",
	ErrInvalidID:   "❌ Invalid arguments, user_id must be a number",
	ErrOperation:   "❌ Operation failed",

	// File-related
	FileNotFound:         "❌ File not found",
	FileNotFoundOrNoPerm: "❌ File not found or permission denied",
	FileSendFailed:       "❌ Failed to send file, it may have been removed from storage",
	FileUnavailable:      "❌ File temporarily unavailable, all storage locations are unreachable",
	FileSaveFailed:       "❌ Failed to save file, please try again later",
	FileAlreadyExists:    "⚠️ File already exists, no need to save again",
	FileDeleteFailed:     "❌ Failed to delete, please try again later",
	FileDeleted:          "✅ File deleted",
	ListFetchFailed:      "❌ Failed to load file list",
	SearchFailed:         "❌ Search failed",
	SearchEmptyQuery:     "❌ Please provide a search keyword, e.g. /search report",
	SearchExpired:        "⏰ Search expired, please search again",
	StatsFetchFailed:     "❌ Failed to load statistics",

	// Random
	RandInvalidNumber: "❌ Invalid count, usage: %[1]s [count], default 5, max %[2]d",
	RandNoMedia:       "❌ No matching media found",
	RandPartial:       "⚠️ Only %[1]d matching item(s) found",
	RandFetchFailed:   "❌ Random fetch failed",

	// User management
	AddUserUsage:         "❌ Invalid arguments, usage: /adduser <user_id>",
	RemoveUserUsage:      "❌ Invalid arguments, usage: /removeuser <user_id>",
	AddUserFailed:        "❌ Failed to add user",
	RemoveUserFailed:     "❌ Failed to remove user",
	DeleteUserFailed:     "❌ Failed to delete user",
	PromoteFailed:        "❌ Promotion failed",
	DemoteFailed:         "❌ Demotion failed",
	UserListFetchFailed:  "❌ Failed to load user list",
	UserNotFound:         "❌ User not found",
	CannotRemoveOwner:    "❌ Cannot remove the Owner",
	CannotModifyOwner:    "❌ Cannot modify the Owner",
	OnlyOwnerRemoveAdmin: "❌ Only the Owner can remove an admin",
	OnlyOwnerDemoteAdmin: "❌ Only the Owner can demote an admin",
	AlreadyAdmin:         "⚠️ User is already an admin",
	AlreadyUser:          "⚠️ User is already a regular user",
	Promoted:             "✅ Promoted to admin",
	Demoted:              "✅ Demoted to regular user",

	UserAlreadyWhitelisted: "⚠️ User %[1]d is already whitelisted",
	UserAdded:              "✅ User %[1]d added to whitelist",
	UserNotInWhitelist:     "❌ User %[1]d is not in the whitelist",
	UserRemoved:            "✅ User %[1]d removed from whitelist",

	// Telegra.ph
	TphUsage:        "❌ Please provide a telegra.ph link, usage: /tph <url>",
	TphInvalidURL:   "❌ Invalid link, only telegra.ph domain is supported",
	TphFetchFailed:  "❌ Failed to fetch article, check the link or try again later",
	TphEmptyContent: "❌ Article is empty",
	TphFetching:     "⏳ Fetching article...",
	TphSaved:        "✅ Saved: %[1]s",

	// Owner maintenance
	EmbBatchStartedRe:       "🚀 /emb_re started (full re-embedding), scanning...",
	EmbBatchStartedSync:     "🚀 /emb_sync started (filling missing embeddings), scanning...",
	EmbBatchBusy:            "⚠️ An embedding batch is already running, please try again later",
	EmbBatchProgress:        "⏳ Progress %[1]d/%[2]d (%[3]s)",
	EmbBatchEmpty:           "ℹ️ Nothing to process",
	EmbBatchDone:            "✅ Embedding batch done: %[1]d files processed",
	EmbBatchFailed:          "❌ Embedding batch failed: %[1]v",
	EmbVectorSearchDisabled: "❌ Vector search is disabled, cannot run embedding batch",

	CapSyncStarted: "🚀 /cap_sync started (filling media group captions)...",
	CapSyncBusy:    "⚠️ A cap_sync task is already running, please try again later",
	CapSyncDone:    "✅ Caption backfill done: %[1]d files updated, %[2]d messages updated\n💡 Captions changed -- consider running /emb_sync to refresh the vector index",
	CapSyncFailed:  "❌ cap_sync failed: %[1]v",

	Welcome: `👋 Welcome to TG Drive Bot!

Send any file / photo / video / audio and I will store and index it for you.

📌 Commands:
/list - Browse my library
/search <keyword> - Search files
/stats - Show statistics
/rand [count] - Random media (default 5, max 10)
/randv [count] - Random videos
/randp [count] - Random photos
/tph <url> - Save a telegra.ph article as Markdown
/lang [zh|en] - View or switch language`,

	WelcomeAdminExtra: `

🛡️ Admin commands:
/adduser <user_id> - Add a user to the whitelist
/removeuser <user_id> - Remove a user from the whitelist
/listuser - Show the user list`,

	WelcomeOwnerExtra: `

👑 Owner commands:
/emb_re - Re-embed all files
/emb_sync - Fill missing embeddings
/cap_sync - Backfill media group captions`,

	// B-class templates
	// media.go
	SavedWithName: "✅ Saved: %[1]s (%[2]s %[3]s, %[4]s)",
	SavedTypeOnly: "✅ Saved: %[1]s %[2]s (%[3]s)",

	// list.go
	ListEmpty:         "📁 My library\n\nNo files stored yet. Just send me a file to get started!",
	ListEmptyType:     "📁 My library\n\nNo files of type %[1]s %[2]s",
	ListHeader:        "📁 My library (%[1]d total)\n\n",
	FileCaptionHeader: "📋 File info\n━━━━━━━━━━━━━━━\n%[1]s Name: %[2]s\n📦 Size: %[3]s\n📅 Saved at: %[4]s\n━━━━━━━━━━━━━━━",

	// search.go
	SearchEmpty:  "🔍 Search results for \"%[1]s\"\n\nNo matches found",
	SearchHeader: "🔍 Search results for \"%[1]s\" (%[2]d total)\n\n",

	StatsBlock: "📊 Your library statistics\n\n",

	// admin_user.go
	UserListHeader:    "👥 User management (%[1]d total)\n\n",
	UserListEmpty:     "👥 User management\n\nNo users yet",
	UserInfoCard:      "👤 User info\n━━━━━━━━━━━━━━━\n📛 Name: %[1]s %[2]s\n🆔 Telegram ID: %[3]d\n🔑 Role: %[4]s\n📦 Files: %[5]d\n📅 Joined: %[6]s",
	BtnBackToUserList: "◀ Back to user list",
	BtnDeleteUser:     "🗑 Delete user",
	BtnPromoteToAdmin: "⬆️ Promote to admin",
	BtnDemoteToUser:   "⬇️ Demote to user",

	// ui/keyboard.go
	BtnDelete:        "🗑 Delete",
	BtnConfirmDelete: "✅ Confirm",
	BtnCancel:        "❌ Cancel",

	// ui/paginator.go
	BtnFilterDocument: "📄 Doc",
	BtnFilterAudio:    "🎵 Audio",
	BtnFilterVideo:    "🎬 Video",
	BtnFilterPhoto:    "🖼 Photo",
	BtnFilterAll:      "All",
	BtnPagePrev:       "◀ Prev",
	BtnPageNext:       "Next ▶",

	// util/format.go
	FileTypeDocument:  "Document",
	FileTypeAudio:     "Audio",
	FileTypeVideo:     "Video",
	FileTypePhoto:     "Photo",
	FileTypeAnimation: "Animation",
	FileTypeVoice:     "Voice",
	FileTypeVideoNote: "Video note",
	FileTypeUnknown:   "Unknown",

	// user_service.go
	RoleAdminName: "Admin",
	RoleUserName:  "User",

	// /lang
	LangUsage:   "Usage: /lang [zh|en]\nView or switch your language.",
	LangCurrent: "🌐 Current language: %[1]s",
	LangSet:     "✅ Language switched to: %[1]s",
	LangInvalid: "❌ Unsupported language, only zh / en are accepted",
}
