// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license

import (
	"context"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

var (
	// ErrLicense is the base error for every license failure.
	ErrLicense = errors.New("license error")
	// ErrMissing is returned while no license applies to this Core.
	ErrMissing = errors.Wrap(ErrLicense, "no license is activated on this Core")
	// ErrExpired is returned while the license on this Core no longer covers it.
	ErrExpired = errors.Wrap(ErrLicense, "the license on this Core has expired")
	// ErrInvalid is returned when a token fails to verify.
	ErrInvalid = errors.Wrap(ErrLicense, "invalid license")
	// ErrFingerprint is returned when the license is bound to other machines.
	ErrFingerprint = errors.Wrap(
		ErrLicense,
		"the license was not issued for this machine",
	)
	// ErrTooMany is returned when a channel would exceed the license's cap.
	ErrTooMany = errors.Wrap(
		ErrLicense,
		"using more channels than allowed by the license",
	)
)

const (
	errorType       = "sy.license"
	missingType     = errorType + ".missing"
	expiredType     = errorType + ".expired"
	invalidType     = errorType + ".invalid"
	fingerprintType = errorType + ".fingerprint"
	tooManyType     = errorType + ".too_many"
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
	if errors.CheapIs(err, ErrFingerprint) {
		return errors.Payload{Type: fingerprintType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrTooMany) {
		return errors.Payload{Type: tooManyType, Data: err.Error()}, true
	}
	if errors.CheapIs(err, ErrLicense) {
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
	case fingerprintType:
		return errors.Wrap(ErrFingerprint, p.Data), true
	case tooManyType:
		return errors.Wrap(ErrTooMany, p.Data), true
	}
	if strings.HasPrefix(p.Type, errorType) {
		return errors.Wrap(ErrLicense, p.Data), true
	}
	return nil, false
}

func init() { errors.Register(encode, decode) }
