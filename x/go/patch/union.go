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

// Discriminator applies the action at c, whose segment names the discriminator of the
// union at dst, to the JSON form of the union. A change of variant keeps the members
// that the new variant shares with the old one, and fails for any other member.
func Discriminator[U any](dst *U, c Cursor) error {
	if !c.Next().Done() {
		return notContainer(c.path[:c.depth+1])
	}
	if c.kind != kindSet && c.kind != kindRemove {
		return c.notList()
	}
	b, err := json.Marshal(*dst)
	if err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: encoding value: %s",
			format(c.path[:c.depth]),
			err,
		)
	}
	var members map[string]any
	if err = json.Unmarshal(b, &members); err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: decoding value: %s",
			format(c.path[:c.depth]),
			err,
		)
	}
	if members == nil {
		if c.kind == kindRemove {
			return nil
		}
		members = map[string]any{}
	}
	if err = c.unguarded(); err != nil {
		return err
	}
	if c.kind == kindRemove {
		delete(members, c.Segment())
	} else if members[c.Segment()], err = normalize(c.value); err != nil {
		return err
	}
	if b, err = json.Marshal(members); err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: encoding value: %s",
			format(c.path[:c.depth]),
			err,
		)
	}
	return decodeJSON(dst, c, b, c.path[:c.depth])
}

// Absent applies the action at c to the union at dst, which holds no variant. A guard
// on the union fails, so the action does nothing. A remove or move does nothing, and
// any other action that goes into the union fails.
func Absent[U any](dst *U, c Cursor, discriminator string) error {
	for _, g := range c.guards {
		if c.names(g.Path, discriminator) {
			return nil
		}
	}
	if c.Segment() == discriminator {
		return Discriminator(dst, c)
	}
	if c.kind == kindRemove || c.kind == kindMove {
		return nil
	}
	return absent(c.path[:c.depth])
}
