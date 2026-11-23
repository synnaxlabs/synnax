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
	// ErrInvalid is returned when a license key fails to verify.
	ErrInvalid = errors.Wrap(ErrLicense, "invalid license")
	// ErrEdition is returned when a Desktop license is activated on a Core that listens
	// on an address other than loopback.
	ErrEdition = errors.Wrap(
		ErrLicense,
		"a Desktop license runs only on a Core that listens on loopback addresses",
	)
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
	editionType     = errorType + ".edition"
	fingerprintType = errorType + ".fingerprint"
	tooManyType     = errorType + ".too_many"
)

func newTooManyError(count uint32) error {
	return errors.Wrapf(ErrTooMany, "limit is %d channels", count)
}

// data returns the message of err without ref. A bare sentinel keeps its own message
// so that clients in other languages receive text.
func data(err, ref error) string {
	if msg := errors.MessageWithout(err, ref); msg != "" {
		return msg
	}
	return err.Error()
}

// wrap decodes data into ref. Data equal to the message of ref is the bare sentinel.
func wrap(ref error, data string) error {
	if data == ref.Error() {
		return ref
	}
	return errors.Wrap(ref, data)
}

func encode(_ *errors.Codec, err error) (errors.Payload, bool) {
	if errors.CheapIs(err, ErrMissing) {
		return errors.Payload{Type: missingType, Data: data(err, ErrMissing)}, true
	}
	if errors.CheapIs(err, ErrExpired) {
		return errors.Payload{Type: expiredType, Data: data(err, ErrExpired)}, true
	}
	if errors.CheapIs(err, ErrInvalid) {
		return errors.Payload{Type: invalidType, Data: data(err, ErrInvalid)}, true
	}
	if errors.CheapIs(err, ErrEdition) {
		return errors.Payload{Type: editionType, Data: data(err, ErrEdition)}, true
	}
	if errors.CheapIs(err, ErrFingerprint) {
		return errors.Payload{
			Type: fingerprintType,
			Data: data(err, ErrFingerprint),
		}, true
	}
	if errors.CheapIs(err, ErrTooMany) {
		return errors.Payload{Type: tooManyType, Data: data(err, ErrTooMany)}, true
	}
	if errors.CheapIs(err, ErrLicense) {
		return errors.Payload{Type: errorType, Data: data(err, ErrLicense)}, true
	}
	return errors.Payload{}, false
}

func decode(_ *errors.Codec, p errors.Payload) (error, bool) {
	switch p.Type {
	case missingType:
		return wrap(ErrMissing, p.Data), true
	case expiredType:
		return wrap(ErrExpired, p.Data), true
	case invalidType:
		return wrap(ErrInvalid, p.Data), true
	case editionType:
		return wrap(ErrEdition, p.Data), true
	case fingerprintType:
		return wrap(ErrFingerprint, p.Data), true
	case tooManyType:
		return wrap(ErrTooMany, p.Data), true
	}
	if strings.HasPrefix(p.Type, errorType) {
		return wrap(ErrLicense, p.Data), true
	}
	return nil, false
}

// ErrorProvider encodes and decodes license errors for an errors.Codec.
func ErrorProvider() errors.Provider {
	return errors.Provider{Encode: encode, Decode: decode}
}
