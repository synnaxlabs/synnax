// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package union provides shared error primitives for discriminated-union types,
// where a typed envelope carries a discriminator alongside variant payloads.
package union

import "github.com/synnaxlabs/x/errors"

// ErrMissingPayload is returned when a discriminated-union envelope names a
// variant via its discriminator but the matching payload is absent. Callers
// match the condition uniformly across packages with errors.Is.
var ErrMissingPayload = errors.New("missing payload for declared variant")

// MissingPayload wraps ErrMissingPayload with the name of the variant whose
// payload was missing for diagnostic context.
func MissingPayload(variant string) error {
	return errors.Wrapf(ErrMissingPayload, "variant %q", variant)
}
