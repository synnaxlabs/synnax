package kv

// Batch  is an ordered collection of key-value operations on the DB. Batch implements
// the Reader interface, and will read key-value pairs from both the Batch and underlying DB.
// A batch must be committed for its changes to be persisted.
type Batch interface {
	Writer
	Reader
	// Close closes the batch without committing it.
	Close() error
	// Commit persists the batch to the underlying DB.
	Commit(opts ...interface{}) error
}
