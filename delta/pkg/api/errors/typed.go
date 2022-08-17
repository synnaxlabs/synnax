package errors

// Type is a classification for an error that occurred under specific conditions. A Type
// gives the caller information on how to parse the accompanied error.
type Type string

// See typed error initializers in errors.go for more info on what each of these
// types mean.
const (
	TypeUnexpected Type = "unexpected"
	TypeGeneral    Type = "general"
	TypeNil        Type = "nil"
	TypeValidation Type = "validation"
	TypeParse      Type = "parse"
	TypeAuth       Type = "auth"
	TypeQuery      Type = "query"
)

// Typed is an error that can be parsed based on its type.
type Typed struct {
	Type Type  `json:"type" msgpack:"type"`
	Err  error `json:"error" msgpack:"error"`
}

// Error implements the error interface.
func (t Typed) Error() string {
	if t.Err != nil {
		return t.Err.Error()
	}
	return "nil"
}

// Occurred returns true if the error is not of type Nil.
func (t Typed) Occurred() bool { return t.Type != TypeNil }
