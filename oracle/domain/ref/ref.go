// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ref reads the @ref domain, which marks a field that holds the keys of other
// resources. A @ref that sets only a direction marks a field above such a field.
package ref

import (
	"slices"

	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

const (
	domainName    = "ref"
	directionExpr = "direction"
)

// Direction is how the holder of a reference uses its targets. Generated Go writes it
// into the Tag of the Gorp reference, so its values never change.
type Direction uint8

const (
	// None is the direction of a reference that declares none.
	None Direction = iota
	// Read is a holder that reads its targets.
	Read
	// Write is a holder that writes its targets.
	Write
)

var directions = map[string]Direction{"read": Read, "write": Write}

// String returns the name of the direction in a schema, or "none".
func (d Direction) String() string {
	switch d {
	case Read:
		return "read"
	case Write:
		return "write"
	case None:
	}
	return "none"
}

// Declares reports whether f holds a reference: it has a @ref that does not set only a
// direction.
func Declares(f resolution.Field) bool {
	d, ok := f.Domains[domainName]
	if !ok {
		return false
	}
	return len(d.Expressions) == 0 || slices.ContainsFunc(
		d.Expressions,
		func(e resolution.Expression) bool { return e.Name != directionExpr },
	)
}

// DirectionOf returns the direction that the @ref of f sets, or None. It returns an
// error when the direction is not one of read or write.
func DirectionOf(f resolution.Field) (Direction, error) {
	d, ok := f.Domains[domainName]
	if !ok {
		return None, nil
	}
	e, ok := d.Expressions.Find(directionExpr)
	if !ok {
		return None, nil
	}
	if len(e.Values) == 1 {
		if dir, ok := directions[e.Values[0].IdentValue]; ok {
			return dir, nil
		}
	}
	return None, errors.Newf(
		"@ref field %s needs direction read or write",
		f.Name,
	)
}
