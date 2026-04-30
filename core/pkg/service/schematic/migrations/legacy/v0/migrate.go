// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Lift decodes the opaque schematic data blob as a v0.Data. v0 is the base of
// the migration chain: any blob that does not announce a recognized newer
// version winds up here and is decoded as v0 on a best-effort basis. A nil
// blob produces a zero Data so empty entries round-trip cleanly.
func Lift(blob msgpack.EncodedJSON) (Data, error) {
	if blob == nil {
		return Data{}, nil
	}
	var d Data
	if err := blob.Unmarshal(&d); err != nil {
		return Data{}, errors.Wrap(err, "decode v0 schematic data")
	}
	return d, nil
}
