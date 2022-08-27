package errors

import (
	"github.com/go-playground/validator/v10"
	"strings"
)

// Field is an error that is associated with a specific field.
type Field struct {
	// Field is the name of the field that caused the error.
	Field string `json:"field" msgpack:"field"`
	// Message is the error Message.
	Message string `json:"message" msgpack:"message"`
}

// Error implements the error interface.
func (f Field) Error() string { return f.Field + ": " + f.Message }

// Fields is an implementation of the error interface that represents a collection of
// field errors.
type Fields []Field

// Error implements the error interface.
func (f Fields) Error() string {
	var s string
	for i, fld := range f {
		s += fld.Error()
		if i != len(f)-1 {
			s += "\n"
		}
	}
	return s
}

func newFieldFromValidator(v validator.FieldError) Field {
	return Field{Field: parseFieldName(v), Message: v.Tag()}
}

// EmbeddedFieldTag can be added to the 'json' or 'msgpack' struct tags on an
// embedded fields so that validation errors do not include the embedded struct
// name as part of the error field name.
const EmbeddedFieldTag = "--embedded--"

func parseFieldName(v validator.FieldError) string {
	// This operation grabs nested struct field names but does not grab the parent
	// struct field name.
	path := strings.Split(v.Namespace(), ".")[1:]

	fieldName := strings.Join(path, ".")
	// and this removes the embedded struct field tag.
	return strings.Replace(fieldName, EmbeddedFieldTag+".", "", -1)
}
