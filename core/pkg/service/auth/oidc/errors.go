// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package oidc

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var (
	errProviderLinked = errors.Wrap(
		validate.ErrValidation,
		"user already has an account of the provider linked",
	)
	errAccountLinked = errors.Wrap(
		validate.ErrValidation,
		"provider account is linked to another user",
	)
)

const (
	providerLinkedType = "sy.validation.provider_linked"
	accountLinkedType  = "sy.validation.account_linked"
)

func encode(_ *errors.Codec, err error) (errors.Payload, bool) {
	data := errors.MessageWithout(err, validate.ErrValidation)
	if errors.CheapIs(err, errProviderLinked) {
		return errors.Payload{Type: providerLinkedType, Data: data}, true
	}
	if errors.CheapIs(err, errAccountLinked) {
		return errors.Payload{Type: accountLinkedType, Data: data}, true
	}
	return errors.Payload{}, false
}

func decode(_ *errors.Codec, p errors.Payload) (error, bool) {
	switch p.Type {
	case providerLinkedType:
		return errProviderLinked, true
	case accountLinkedType:
		return errAccountLinked, true
	}
	return nil, false
}

// ErrorProvider encodes and decodes the validation errors that refuse a link, as types
// sy.validation.provider_linked and sy.validation.account_linked. It must come before
// the validation provider, which would encode them as plain validation errors.
func ErrorProvider() errors.Provider {
	return errors.Provider{Encode: encode, Decode: decode}
}
