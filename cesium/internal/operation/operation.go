package operation

import (
	"context"
	"github.com/arya-analytics/x/kfs"
)

type Operation[F comparable] interface {
	// Context returns a context, that when canceled represents a forced abort of the operation.
	Context() context.Context
	// FileKey returns the key of the file to which the operation applies.
	FileKey() F
	// WriteError sends an error to the operation. This is only used for IO errors.
	WriteError(error)
	// Exec is called by Persist to execute the operation. The provided file will have the key returned by FileKey.
	// The operation has a lock on the file during this time, and is free to make any modifications.
	Exec(f kfs.File[F])
}

type Set[F comparable, O Operation[F]] []O

func (s Set[F, T]) Context() context.Context {
	return s[0].Context()
}

func (s Set[F, T]) FileKey() F {
	return s[0].FileKey()
}

func (s Set[F, T]) Exec(f kfs.File[F]) {
	for _, op := range s {
		op.Exec(f)
	}
}

func (s Set[F, T]) WriteError(err error) {
	for _, op := range s {
		op.WriteError(err)
	}
}
