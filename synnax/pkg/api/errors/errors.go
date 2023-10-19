// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package errors implements standard error types and formats to provide uniform,
// parseable exceptions to delta API consumers.
//
// All errors in this package are variations of the Typed struct; an error's type gives
// the API consumer information on how it should be parsed. All errors in this package
// can be encoded, and implement freighter interfaces for go-internal and cross language
// parsing.
package errors

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"strings"
)

const prefix = "sy.api."

// See typed error initializers in errors.go for more info on what each of these
// types mean.
const (
	TypeUnexpected ferrors.Type = prefix + "unexpected"
	TypeGeneral    ferrors.Type = prefix + "general"
	TypeNil        ferrors.Type = prefix + "nil"
	TypeValidation ferrors.Type = prefix + "validation"
	TypeParse      ferrors.Type = prefix + "parse"
	TypeAuth       ferrors.Type = prefix + "auth"
	TypeQuery      ferrors.Type = prefix + "query"
	TypeRoute      ferrors.Type = prefix + "route"
)

// General is an error that doesn't fit into a specific t. General errors should be
// used in the case where an error is expected to occur during normal use, such as a
// query returning multiple results when it should return exactly one. Unexpected errors
// should be used in the case where an error should not
// occur during normal use.
func General(err error) ferrors.Error { return ferrors.Typed(err, TypeGeneral) }

// Unexpected represents an error that should not occur during normal execution.
// For example, a seemingly healthy node is unreachable or a transaction fails
// to commit.
func Unexpected(err error) ferrors.Error { return ferrors.Typed(err, TypeUnexpected) }

// Parse is an error that occurs when a user-supplied value cannot be parsed.
// For example, a user sends msgpack data when it should be JSON.
func Parse(err error) ferrors.Error { return ferrors.Typed(err, TypeParse) }

// Auth is an error that occurs when a user is not authorized to perform an
// action or when authorization fails.
func Auth(err error) ferrors.Error { return ferrors.Typed(err, TypeAuth) }

// Query is an error that occurs when a particular query fails.
func Query(err error) ferrors.Error { return ferrors.Typed(err, TypeQuery) }

type routeError struct {
	Path string `json:"path" msgpack:"path"`
}

func (r routeError) Error() string { return r.Path }

func Route(err error, path string) ferrors.Error {
	return ferrors.Typed(routeError{Path: path}, TypeRoute)
}

func Auto(err error) ferrors.Error {
	if err == nil {
		return ferrors.Nil
	}
	var t ferrors.Error
	if errors.As(err, &t) {
		return t
	}
	if errors.Is(err, query.Error) {
		return Query(err)
	}
	if errors.Is(err, validate.Error) {
		return Validation(err)
	}
	return General(err)
}

// Validation is an error that occurs when a user-supplied value is invalid. For example,
// a user provides an email address without the	@ symbol. Validation attempts to parse
// validation errors from the validator package into a set of Field errors.
func Validation(err error) ferrors.Error {
	if err == nil {
		return ferrors.Nil
	}
	var fields Fields
	if errors.As(err, &fields) {
		return fields
	}
	var field Field
	if errors.As(err, &field) {
		return field
	}
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		if errors.Is(err, validate.Error) {
			return ferrors.Typed(err, TypeValidation)
		}
		return Unexpected(err)
	}
	for _, e := range validationErrors {
		fields = append(fields, newFieldFromValidator(e))
	}
	return fields
}

// Field is an error that is associated with a specific field.
type Field struct {
	// Field is the name of the field that caused the error.
	Field string `json:"field" msgpack:"field"`
	// Message is the error Message.
	Message string `json:"message" msgpack:"message"`
}

func (f Field) FreighterType() ferrors.Type { return TypeValidation }

// Error implements the error interface.
func (f Field) Error() string { return f.Field + ": " + f.Message }

// Fields is an implementation of the error interface that represents a collection of
// field errors.
type Fields []Field

func (f Fields) FreighterType() ferrors.Type { return TypeValidation }

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
const EmbeddedFieldTag = "--embed--"

func parseFieldName(v validator.FieldError) string {
	// This operation grabs nested struct field names but does not grab the parent
	// struct field name.
	path := strings.Split(v.Namespace(), ".")[1:]

	fieldName := strings.Join(path, ".")
	// and this removes the embedded struct field tag.
	return strings.Replace(fieldName, EmbeddedFieldTag+".", "", -1)
}

var ecd = &binary.JSONEncoderDecoder{}

func encode(_ context.Context, err error) (p ferrors.Payload, ok bool) {
	var tErr ferrors.Error
	if !errors.As(err, &tErr) || !strings.HasPrefix(string(tErr.FreighterType()), prefix) {
		return
	}
	return ferrors.Payload{Type: tErr.FreighterType(), Data: err.Error()}, true
}
func decode(ctx context.Context, pld ferrors.Payload) (error, bool) {
	if !strings.HasPrefix(string(pld.Type), prefix) {
		return nil, false
	}
	switch pld.Type {
	case TypeValidation:
		return parseValidationError(ctx, pld.Data), true
	default:
		return errors.New(pld.Data), true
	}
}

func init() { ferrors.Register(encode, decode) }

func parseValidationError(ctx context.Context, data string) error {
	var raw []interface{}
	lo.Must0(ecd.Decode(ctx, []byte(data), &raw))
	var fields Fields
	for _, rawField := range raw {
		fieldMap, ok := rawField.(map[string]interface{})
		if !ok {
			panic("[freighter] - invalid error message")
		}
		fields = append(fields, Field{
			Field:   fieldMap["field"].(string),
			Message: fieldMap["message"].(string),
		})
	}
	return Validation(fields)
}

func Middleware() freighter.Middleware {
	return freighter.MiddlewareFunc(func(ctx freighter.Context, next freighter.Next) (freighter.Context, error) {
		oCtx, err := next(ctx)
		if err == nil {
			return oCtx, nil
		}
		return oCtx, Auto(err)
	})
}
