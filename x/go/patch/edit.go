// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch

import (
	"encoding/json/v2"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Edit applies fn to the JSON form of doc and decodes the result into a new D. The
// decode rejects members that D does not have, so an action outside D fails. Edit
// returns the error of fn, or an error wrapping validate.ErrValidation when doc does
// not encode or the result does not decode.
func Edit[D any](doc D, fn func(map[string]any) error) (D, error) {
	var zero D
	b, err := json.Marshal(doc)
	if err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "encoding document: %s", err)
	}
	var m map[string]any
	if err = json.Unmarshal(b, &m); err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "decoding document: %s", err)
	}
	if err = fn(m); err != nil {
		return zero, err
	}
	if b, err = json.Marshal(m); err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "encoding document: %s", err)
	}
	var next D
	if err = json.Unmarshal(b, &next, json.RejectUnknownMembers(true)); err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "decoding document: %s", err)
	}
	return next, nil
}
