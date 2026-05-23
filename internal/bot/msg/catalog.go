// Package msg centralizes user-facing text so handlers don't embed strings inline.
//
// Localized copies are exposed as Catalog instances (Zh, En) and selected per
// request by middleware via msg.For(lang). Format strings in this file MUST use
// indexed placeholders ("%[1]s", "%[2]d", ...) so that translations can reorder
// arguments without changing call sites.
package msg

// Catalog holds every user-facing string for a single language. Keep field
// names stable across languages -- only the values differ between Zh and En.
type Catalog struct {
	// ----- Generic errors -----
	ErrUnknown     string
	ErrInvalidArgs string
	ErrInvalidID   string
	ErrOperation   string

	// ----- File-related messages -----
	FileNotFound         string
	FileNotFoundOrNoPerm string
	FileSendFailed       string
	FileUnavailable      string
	FileSaveFailed       string
	FileAlreadyExists    string
	FileDeleteFailed     string
	FileDeleted          string
	ListFetchFailed      string
	SearchFailed         string
	SearchEmptyQuery     string
	SearchExpired        string
	StatsFetchFailed     string

	// ----- Random media -----
	RandInvalidNumber string // "%[1]s [count], default 5, max %[2]d"
	RandNoMedia       string
	RandPartial       string // "only %[1]d found"
	RandFetchFailed   string

	// ----- User management -----
	AddUserUsage         string
	RemoveUserUsage      string
	AddUserFailed        string
	RemoveUserFailed     string
	DeleteUserFailed     string
	PromoteFailed        string
	DemoteFailed         string
	UserListFetchFailed  string
	UserNotFound         string
	CannotRemoveOwner    string
	CannotModifyOwner    string
	OnlyOwnerRemoveAdmin string
	OnlyOwnerDemoteAdmin string
	AlreadyAdmin         string
	AlreadyUser          string
	Promoted             string
	Demoted              string

	// admin_user.go inline 模板
	UserAlreadyWhitelisted string // "%[1]d"
	UserAdded              string // "%[1]d"
	UserNotInWhitelist     string // "%[1]d"
	UserRemoved            string // "%[1]d"

	// Telegra.ph
	TphUsage        string
	TphInvalidURL   string
	TphFetchFailed  string
	TphEmptyContent string
	TphFetching     string
	TphSaved        string // "%[1]s"

	// Owner maintenance
	EmbBatchStartedRe       string
	EmbBatchStartedSync     string
	EmbBatchBusy            string
	EmbBatchProgress        string // "%[1]d/%[2]d (%[3]s)"
	EmbBatchEmpty           string
	EmbBatchDone            string // "%[1]d"
	EmbBatchFailed          string // "%[1]v"
	EmbVectorSearchDisabled string

	CapSyncStarted string
	CapSyncBusy    string
	CapSyncDone    string // "%[1]d files / %[2]d messages"
	CapSyncFailed  string // "%[1]v"

	// Start banner
	Welcome           string
	WelcomeAdminExtra string
	WelcomeOwnerExtra string

	// ----- B-class: format templates lifted from handlers -----
	// media.go
	SavedWithName string // "%[1]s %[2]s %[3]s %[4]s" -- name, icon, typeName, size
	SavedTypeOnly string // "%[1]s %[2]s %[3]s"       -- icon, typeName, size

	// list.go
	ListEmpty         string                                  // empty library banner
	ListEmptyType     string // "no %[1]s %[2]s type"          -- icon, typeName
	ListHeader        string // "(%[1]d total)"                -- total
	FileCaptionHeader string // "%[1]s %[2]s %[3]s %[4]s"      -- icon, fileName, size, createdAt

	// search.go
	SearchEmpty  string // "%[1]s"                              -- query
	SearchHeader string // "%[1]s %[2]d"                        -- query, total

	// stats.go (rendered as a single multi-line block; see zh/en for the exact template)
	StatsBlock string // big multi-arg template; placeholders documented in zh.go

	// admin_user.go -- user list/info templates and buttons
	UserListHeader     string // "(%[1]d users)"
	UserListEmpty      string
	UserInfoCard       string // multi-arg; see zh.go
	BtnBackToUserList  string
	BtnDeleteUser      string
	BtnPromoteToAdmin  string
	BtnDemoteToUser    string

	// ui/keyboard.go -- file action buttons
	BtnDelete        string
	BtnConfirmDelete string
	BtnCancel        string

	// ui/paginator.go -- filter / pagination buttons
	BtnFilterDocument string
	BtnFilterAudio    string
	BtnFilterVideo    string
	BtnFilterPhoto    string
	BtnFilterAll      string
	BtnPagePrev       string
	BtnPageNext       string

	// util/format.go -- file type display names
	FileTypeDocument  string
	FileTypeAudio     string
	FileTypeVideo     string
	FileTypePhoto     string
	FileTypeAnimation string
	FileTypeVoice     string
	FileTypeVideoNote string
	FileTypeUnknown   string

	// user_service.go -- role display names
	RoleAdminName string
	RoleUserName  string

	// /lang command
	LangUsage   string
	LangCurrent string // "%[1]s"
	LangSet     string // "%[1]s"
	LangInvalid string
}
