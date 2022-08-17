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

// Fields is an implementation of the error interface that represents a collection of field errors.
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

func parseFieldName(v validator.FieldError) string {
	// This operation grabs nested struct field names but does not grab the parent
	// struct field name.
	path := strings.Split(v.Namespace(), ".")[1:]

	// Rejoin the path with dots.
	fieldName := strings.Join(path, ".")

	// We use the json tag '.' to indicate an embedded struct,
	// and this removes the embedded struct field name.
	if len(fieldName) > 0 {
		for {
			if fieldName[0] != '.' {
				break
			}
			fieldName = fieldName[1:]
		}
	}
	return fieldName
}
