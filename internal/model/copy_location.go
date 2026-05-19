package model

// CopyLocation identifies a single stored copy of a file: which channel it lives in and which
// message id it was written as. Used in-memory between the storage layer and the service layer
// when forwarding, retrieving, or deleting copies. Persisted form lives in the file_copies table
// (see FileCopy).
type CopyLocation struct {
	ChatID int64
	MsgID  int
}
