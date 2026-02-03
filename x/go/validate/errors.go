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
	// ErrValidation is a base class for all validation errors. Validation errors should
	// be returned when caller provided data (whether user or internal) fails to meet
	// specific rules or criteria defined to ensure its correctness, completeness, or
	// format.
	ErrValidation  = errors.New("validation error")
	ErrInvalidType = errors.Wrapf(ErrValidation, "invalid type")
	ErrConversion  = errors.Wrap(ErrValidation, "conversion error")
	ErrRequired    = errors.Wrap(ErrValidation, "required")
)

type PathError struct {
	Err  error    `json:"error"`
	Path []string `json:"path"`
}

func (p PathError) joinPath() string {
	return strings.Join(p.Path, ".")
}

type encodedPathError struct {
	Error errors.Payload `json:"error"`
	Path  []string       `json:"path"`
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
	var errPath PathError
	segments := pathToSegments(path...)
	if errors.As(err, &errPath) {
		errPath.Path = append(segments, errPath.Path...)
	} else {
		errPath.Path = segments
		errPath.Err = err
	}
	return errPath
}

func (p PathError) Error() string { return p.joinPath() + ": " + p.Err.Error() }

func NewInvalidTypeError(expected, received string) error {
	return errors.Wrapf(ErrInvalidType, "expected %s but received %s", expected, received)
}

const (
	baseErrorType = "sy.validation"
	pathErrorType = baseErrorType + ".path"
)

func encode(ctx context.Context, err error) (errors.Payload, bool) {
	var errPath PathError
	if errors.As(err, &errPath) {
		internal := errors.Encode(ctx, errPath.Err, false)
		return errors.Payload{
			Type: pathErrorType,
			Data: string(lo.Must(json.Marshal(encodedPathError{
				Error: internal,
				Path:  errPath.Path,
			}))),
		}, true
	}
	if errors.Is(err, ErrValidation) {
		return errors.Payload{Type: "sy.validation", Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(ctx context.Context, p errors.Payload) (error, bool) {
	if !strings.HasPrefix(p.Type, baseErrorType) {
		return nil, false
	}
	if p.Type == pathErrorType {
		var errDecodedPath encodedPathError
		if err := json.Unmarshal([]byte(p.Data), &errDecodedPath); err != nil {
			return err, true
		}
		return PathError{
			Path: errDecodedPath.Path,
			Err:  errors.Decode(ctx, errDecodedPath.Error),
		}, true
	}
	return errors.Wrapf(ErrValidation, p.Data), true
}

func init() { errors.Register(encode, decode) }
