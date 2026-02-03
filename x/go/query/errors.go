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
	ErrQuery = errors.New("query")
	// ErrNotFound is returned when a requested entity cannot be found.
	ErrNotFound = errors.Wrap(ErrQuery, "not found")
	// ErrUniqueViolation is returned when a unique constraint on a particular entity is
	// violated.
	ErrUniqueViolation = errors.Wrap(ErrQuery, "unique violation")
	// ErrInvalidParameters is returned when a query has invalid parameters.
	ErrInvalidParameters = errors.Wrap(ErrQuery, "invalid parameters")
)

const (
	typePrefix        = "sy.query"
	uniqueViolation   = typePrefix + ".unique_violation"
	notFound          = typePrefix + ".not_found"
	invalidParameters = typePrefix + ".invalid_parameters"
)

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, ErrNotFound) {
		return errors.Payload{Type: notFound, Data: err.Error()}, true
	}
	if errors.Is(err, ErrUniqueViolation) {
		return errors.Payload{Type: uniqueViolation, Data: err.Error()}, true
	}
	if errors.Is(err, ErrInvalidParameters) {
		return errors.Payload{Type: invalidParameters, Data: err.Error()}, true
	}
	if errors.Is(err, ErrQuery) {
		return errors.Payload{Type: typePrefix, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, pld errors.Payload) (error, bool) {
	switch pld.Type {
	case notFound:
		return errors.Wrap(ErrNotFound, pld.Data), true
	case uniqueViolation:
		return errors.Wrap(ErrUniqueViolation, pld.Data), true
	case invalidParameters:
		return errors.Wrap(ErrInvalidParameters, pld.Data), true
	}
	if strings.HasPrefix(pld.Type, typePrefix) {
		return errors.Wrap(ErrQuery, pld.Data), true
	}
	return nil, false
}

func init() {
	errors.Register(encode, decode)
}
