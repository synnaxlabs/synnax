// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"context"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

var (
	// ErrVerification is the base error for every verification failure.
	ErrVerification = errors.New("license error")
	// ErrMissing is returned while no grant applies to this Core.
	ErrMissing = errors.Wrap(ErrVerification, "no license is activated on this Core")
	// ErrExpired is returned while the grant on this Core no longer covers it.
	ErrExpired = errors.Wrap(ErrVerification, "the license on this Core has expired")
	// ErrInvalid is returned when a token fails to verify.
	ErrInvalid = errors.Wrap(ErrVerification, "invalid license")
	// ErrHost is returned when a grant is bound to hosts this machine is not one of.
	ErrHost = errors.Wrap(
		ErrVerification,
		"the license was not issued for this machine",
	)
	// ErrTooMany is returned when a channel would exceed the grant's cap.
	ErrTooMany = errors.Wrap(
		ErrVerification,
		"using more channels than allowed by the license",
	)
)

const (
	errorType   = "sy.verification"
	missingType = errorType + ".missing"
	expiredType = errorType + ".expired"
	invalidType = errorType + ".invalid"
	hostType    = errorType + ".host"
	tooManyType = errorType + ".too_many"
)

const errTooManyWrapString = "limit is %d channels"

func newTooManyError(count uint32) error {
	return errors.Wrapf(ErrTooMany, errTooManyWrapString, count)
}

func encode(_ context.Context, err error) (errors.Payload, bool) {
	if errors.CheapIs(err, ErrMissing) {
		return errors.Payload{Type: missingType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrExpired) {
		return errors.Payload{Type: expiredType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrInvalid) {
		return errors.Payload{Type: invalidType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrHost) {
		return errors.Payload{Type: hostType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrTooMany) {
		return errors.Payload{Type: tooManyType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrVerification) {
		return errors.Payload{Type: errorType, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decode(_ context.Context, p errors.Payload) (error, bool) {
	switch p.Type {
	case missingType:
		return errors.Wrap(ErrMissing, p.Data), true
	case expiredType:
		return errors.Wrap(ErrExpired, p.Data), true
	case invalidType:
		return errors.Wrap(ErrInvalid, p.Data), true
	case hostType:
		return errors.Wrap(ErrHost, p.Data), true
	case tooManyType:
		return errors.Wrap(ErrTooMany, p.Data), true
	}
	if strings.HasPrefix(p.Type, errorType) {
		return errors.Wrap(ErrVerification, p.Data), true
	}
	return nil, false
}

func init() { errors.Register(encode, decode) }
