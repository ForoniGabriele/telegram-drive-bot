package constants

// FileType is a supported media type identifier used in DB rows and callback data.
type FileType string

const (
	FileTypeAll       FileType = "all"
	FileTypeDocument  FileType = "document"
	FileTypeAudio     FileType = "audio"
	FileTypeVideo     FileType = "video"
	FileTypePhoto     FileType = "photo"
	FileTypeAnimation FileType = "animation"
	FileTypeVoice     FileType = "voice"
	FileTypeVideoNote FileType = "video_note"
)

func (t FileType) String() string { return string(t) }

// DateShort is used inside list/search result rows.
const DateShort = "2006.1.2"

// DateFull is used in file-detail captions.
const DateFull = "2006-01-02 15:04"

// DateDay is used in user-info and stats views.
const DateDay = "2006-01-02"
