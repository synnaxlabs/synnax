// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query

import (
	"context"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

var (
	Error = errors.New("query")
	// NotFound is returned when a requested entity cannot be found.
	NotFound = errors.Wrap(Error, "not found")
	// UniqueViolation is returned when a unique constraint on a particular entity
	// is violated.
	UniqueViolation = errors.Wrap(Error, "unique violation")
	// InvalidParameters is returned when a query has invalid parameters.
	InvalidParameters = errors.Wrap(Error, "invalid parameters")
)

const (
	typePrefix        = "sy.query"
	uniqueViolation   = typePrefix + ".unique_violation"
	notFound          = typePrefix + ".not_found"
	invalidParameters = typePrefix + ".invalid_parameters"
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, NotFound) {
		return errors.Payload{Type: notFound, Data: err.Error()}, true
	}
	if errors.Is(err, UniqueViolation) {
		return errors.Payload{Type: uniqueViolation, Data: err.Error()}, true
	}
	if errors.Is(err, InvalidParameters) {
		return errors.Payload{Type: invalidParameters, Data: err.Error()}, true
	}
	if errors.Is(err, Error) {
		return errors.Payload{Type: typePrefix, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, pld errors.Payload) (error, bool) {
	switch pld.Type {
	case notFound:
		return errors.Wrap(NotFound, pld.Data), true
	case uniqueViolation:
		return errors.Wrap(UniqueViolation, pld.Data), true
	case invalidParameters:
		return errors.Wrap(InvalidParameters, pld.Data), true
	}
	if strings.HasPrefix(pld.Type, typePrefix) {
		return errors.Wrap(Error, pld.Data), true
	}
	return nil, false
}

func init() {
	errors.Register(encode, decode)
}
