// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/errors"

// IsRead returns true if the direction includes read.
func (d ChanDirection) IsRead() bool { return d&ChanDirectionRead != 0 }

// IsWrite returns true if the direction includes write.
func (d ChanDirection) IsWrite() bool { return d&ChanDirectionWrite != 0 }

// IsSet returns true if any direction has been specified.
func (d ChanDirection) IsSet() bool { return d != ChanDirectionNone }

// CheckCompatibility returns an error if actual is incompatible with the
// required direction d. Returns nil when compatible or when either side
// has no direction set.
func (d ChanDirection) CheckCompatibility(actual ChanDirection) error {
	if !d.IsSet() || !actual.IsSet() {
		return nil
	}
	if d.IsWrite() && !actual.IsWrite() {
		return errors.New("expected a write channel, but got a read channel")
	}
	if d.IsRead() && !actual.IsRead() {
		return errors.New("expected a read channel, but got a write channel")
	}
	return nil
}
