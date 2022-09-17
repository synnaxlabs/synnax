package ferrors

type Type string

const (
	// Empty represents an error that hasn't been properly parsed or detected.
	// Empty errors typically represent a programming error.
	Empty = Type("")
	// Nil represents a nil error i.e. one that has not occurred.
	Nil = Type("nil")
	// Unknown represents an error that was not registered with the ferrors package.
	Unknown = Type("unknown")
	// Roach represents an error type that was encoded using cockroachdb's errors package.
	// This is the default error type for errors that are not registered with the ferrors package,
	// and is used mostly for go-to-go communication.
	Roach = Type("roach")
)

type Error interface {
	error
	// FreighterType returns the type of the error. Freighter uses this to determine the
	// correct decode to use on the other end of the transport.
	FreighterType() Type
}

// Typed is the easiest way to create an error type that satisfies the Error interface.
// It takes the error and attaches the provided type to it. Then you can define
// encode and decode functions for the type and register them with the ferrors package.
func Typed(err error, t Type) error {
	return &typed{error: err, t: t}
}

type typed struct {
	error
	t Type
}

var _ Error = (*typed)(nil)

// FreighterType implements Error.
func (t typed) FreighterType() Type { return t.t }
