package errors

import (
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
)

// Nil is a typed representation of a nil error.
var Nil = Typed{Type: TypeNil}

var Cancelled = Typed{Type: TypeGeneral, Err: Message{Message: "request cancelled"}}

type Message struct {
	Message string `json:"message" msgpack:"message"`
}

func (g Message) Error() string { return g.Message }

func newTypedMessage(t Type, msg string) Typed {
	return Typed{Type: t, Err: Message{Message: msg}}
}

// General is an error that doesn't fit into a specific Type. General errors
// should be used in the case where an error is expected to occur during normal
// use, such as a query returning multiple results when it should return exactly
// one. Unexpected errors should be used in the case where an error should not
// occur during normal use.
func General(err error) Typed { return newTypedMessage(TypeGeneral, err.Error()) }

// MaybeGeneral is a convenience function for returning a General error if the
// error is not nil.
func MaybeGeneral(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return General(err)
}

// Unexpected represents an error that should not occur during normal execution.
// For example, a seemingly healthy node is unreachable or a transaction fails
// to commit.
func Unexpected(err error) Typed { return newTypedMessage(TypeUnexpected, err.Error()) }

// MaybeUnexpected is a convenience function for returning a Unexpected error if
// the error is not nil.
func MaybeUnexpected(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Unexpected(err)
}

// Parse is an error that occurs when a user-supplied value cannot be parsed.
// For example, a user sends msgpack data when it should be JSON.
func Parse(err error) Typed { return newTypedMessage(TypeParse, err.Error()) }

// MaybeParse is a convenience function for returning a Parse error if the error
// is not nil.
func MaybeParse(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Parse(err)
}

// Auth is an error that occurs when a user is not authorized to perform an
// action or when authorization fails.
func Auth(err error) Typed { return newTypedMessage(TypeAuth, err.Error()) }

// MaybeAuth is a convenience function for returning a Auth error if the error
// is not nil.
func MaybeAuth(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Auth(err)
}

// Query is an error that occurs when a particular query fails.
func Query(err error) Typed {
	if errors.Is(err, query.NotFound) {
		return newTypedMessage(TypeQuery, err.Error())
	}
	if errors.Is(err, query.UniqueViolation) {
		return newTypedMessage(TypeQuery, err.Error())
	}
	return General(err)
}

// MaybeQuery is a convenience function for returning a Query error if the error
// is not nil.
func MaybeQuery(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Query(err)
}

func maybe(err error) (Typed, bool) {
	if err == nil {
		return Nil, true
	}
	if t, ok := err.(Typed); ok {
		return t, true
	}
	return Typed{}, false
}
