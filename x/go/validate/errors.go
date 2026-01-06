// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
)

var (
	// Error is a base class for all validation errors. Validation errors should be returned
	// when caller provided data (whether user or internal) fails to meet specific rules
	// or criteria defined to ensure its correctness, completeness, or format.
	Error            = errors.New("validation error")
	InvalidTypeError = errors.Wrapf(Error, "invalid type")
	ConversionError  = errors.Wrap(Error, "conversion error")
	RequiredError    = errors.Wrap(Error, "required")
)

type PathError struct {
	Path []string `json:"path"`
	Err  error    `json:"error"`
}

func (p PathError) joinPath() string {
	return strings.Join(p.Path, ".")
}

type encodedPathError struct {
	Path  []string       `json:"path"`
	Error errors.Payload `json:"error"`
}

func pathToSegments(segments ...string) []string {
	if len(segments) == 1 {
		return lo.Map(strings.Split(segments[0], "."), func(s string, _ int) string {
			return lo.SnakeCase(s)
		})
	}
	return lo.FlatMap(segments, func(s string, _ int) []string {
		return pathToSegments(s)
	})
}

func PathedError(err error, path ...string) error {
	var pathErr PathError
	segments := pathToSegments(path...)
	if errors.As(err, &pathErr) {
		pathErr.Path = append(segments, pathErr.Path...)
	} else {
		pathErr.Path = segments
		pathErr.Err = err
	}
	return pathErr
}

func (p PathError) Error() string { return p.joinPath() + ": " + p.Err.Error() }

func NewInvalidTypeError(expected, received string) error {
	return errors.Wrapf(InvalidTypeError, "expected %s but received %s", expected, received)
}

const (
	baseErrorType = "sy.validation"
	pathErrorType = baseErrorType + ".path"
)

func encode(ctx context.Context, err error) (errors.Payload, bool) {
	var fe PathError
	if errors.As(err, &fe) {
		internal := errors.Encode(ctx, fe.Err, false)
		return errors.Payload{
			Type: pathErrorType,
			Data: string(lo.Must(json.Marshal(encodedPathError{
				Error: internal,
				Path:  fe.Path,
			}))),
		}, true
	}
	if errors.Is(err, Error) {
		return errors.Payload{Type: "sy.validation", Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(ctx context.Context, p errors.Payload) (error, bool) {
	if !strings.HasPrefix(p.Type, baseErrorType) {
		return nil, false
	}
	if p.Type == pathErrorType {
		var decodedPathError encodedPathError
		if err := json.Unmarshal([]byte(p.Data), &decodedPathError); err != nil {
			return err, true
		}
		return PathError{
			Path: decodedPathError.Path,
			Err:  errors.Decode(ctx, decodedPathError.Error),
		}, true
	}
	return errors.Wrapf(Error, p.Data), true
}

func init() { errors.Register(encode, decode) }
