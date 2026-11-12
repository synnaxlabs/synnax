// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import (
	"bytes"
	"encoding/base64"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Generator generates the value of a uuid or timestamp write field for each payload.
type Generator struct {
	timestamp bool
	format    xjson.TimeFormat
}

// Append appends the JSON value generated at now to dst.
func (g Generator) Append(dst []byte, now telem.TimeStamp) ([]byte, error) {
	if g.timestamp {
		return xjson.AppendTimeStamp(dst, now, g.format), nil
	}
	dst = append(dst, '"')
	dst, err := uuid.New().AppendText(dst)
	return append(dst, '"'), err
}

// ValidateWriteField returns an error wrapping validate.ErrValidation when f is null or
// its value cannot go into a payload: a float that is not finite, json text that is not
// JSON, or bytes text that is not base64.
func ValidateWriteField(f WriteField) error {
	if f.Variant == nil {
		return errors.Wrap(validate.ErrValidation, "a field is null")
	}
	_, err := literal(f)
	return err
}

// TemplateField returns the template field of f, labeled by its key. Its Literal is
// the value f places in every payload. A uuid or timestamp field has no Literal, and
// gen generates its value. It returns an error wrapping validate.ErrValidation when f
// has no pointer or ValidateWriteField fails.
func TemplateField(f WriteField) (xjson.TemplateField, Generator, error) {
	if f.Variant == nil {
		return xjson.TemplateField{}, Generator{}, errors.Wrap(
			validate.ErrValidation, "a field is null",
		)
	}
	field := xjson.TemplateField{Label: fmt.Sprintf("field %q", f.Base().Key)}
	fail := func(err error) (xjson.TemplateField, Generator, error) {
		return xjson.TemplateField{}, Generator{}, errors.Wrap(err, field.Label)
	}
	var (
		gen Generator
		err error
	)
	if field.Pointer, err = xjson.ParsePointer(f.Base().Pointer); err != nil {
		return fail(err)
	}
	if len(field.Pointer) == 0 {
		return fail(errors.Wrap(validate.ErrValidation, "pointer must not be empty"))
	}
	switch v := f.Variant.(type) {
	case UuidWriteField:
	case TimestampWriteField:
		gen.timestamp = true
		if gen.format, err = xjson.ParseTimeFormat(string(v.TimeFormat)); err != nil {
			return fail(err)
		}
	default:
		if field.Literal, err = literal(f); err != nil {
			return fail(err)
		}
	}
	return field, gen, nil
}

// literal returns the JSON text of the value of f, nil for a uuid or timestamp field.
func literal(f WriteField) ([]byte, error) {
	switch v := f.Variant.(type) {
	case FloatWriteField:
		if math.IsNaN(float64(v.Value)) || math.IsInf(float64(v.Value), 0) {
			return nil, errors.Wrapf(
				validate.ErrValidation, "value %v must be finite", float64(v.Value),
			)
		}
		return xjson.Marshal(float64(v.Value))
	case IntegerWriteField:
		return strconv.AppendInt(nil, int64(v.Value), 10), nil
	case UnsignedWriteField:
		return strconv.AppendUint(nil, uint64(v.Value), 10), nil
	case StringWriteField:
		return jsontext.AppendQuote(nil, v.Value)
	case BooleanWriteField:
		return strconv.AppendBool(nil, v.Value), nil
	case JsonWriteField:
		value := jsontext.Value(v.Value)
		if !isJSON(value) {
			return nil, errors.Wrap(validate.ErrValidation, "value is not JSON")
		}
		if err := value.Compact(); err != nil {
			return nil, err
		}
		return value, nil
	case BytesWriteField:
		// The strict decoder still skips line breaks, which the Console rejects.
		if strings.ContainsAny(v.Value, "\r\n") {
			return nil, errors.Wrap(validate.ErrValidation, "value is not base64")
		}
		if _, err := base64.StdEncoding.Strict().DecodeString(v.Value); err != nil {
			return nil, errors.Wrapf(
				validate.ErrValidation, "value is not base64: %v", err,
			)
		}
		return jsontext.AppendQuote(nil, v.Value)
	default:
		return nil, nil
	}
}

// isJSON reports whether the Console and the Driver accept value as JSON: duplicate
// names pass, and a lone surrogate or a number past float64 range fails.
func isJSON(value jsontext.Value) bool {
	if !value.IsValid(jsontext.AllowDuplicateNames(true)) {
		return false
	}
	dec := jsontext.NewDecoder(
		bytes.NewReader(value),
		jsontext.AllowDuplicateNames(true),
	)
	for {
		tok, err := dec.ReadToken()
		if errors.Is(err, io.EOF) {
			return true
		}
		if err != nil {
			return false
		}
		if tok.Kind() == '0' {
			if f, err := strconv.ParseFloat(tok.String(), 64); err != nil &&
				math.IsInf(f, 0) {
				return false
			}
		}
	}
}
