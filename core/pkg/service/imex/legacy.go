// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// PeekVersion extracts the Version stamped inside an opaque legacy data blob. A nil
// blob and a blob without a version field both report version 0, so callers dispatch
// them to the earliest legacy shape. resource names the blob in errors ("log data").
func PeekVersion(blob msgpack.EncodedJSON, resource string) (Version, error) {
	if blob == nil {
		return 0, nil
	}
	var peek struct {
		Version Version `json:"version"`
	}
	if err := blob.Unmarshal(&peek); err != nil {
		return 0, errors.Wrapf(err, "peek %s version", resource)
	}
	return peek.Version, nil
}

// DecodeBlob unmarshals blob as T, treating a nil blob as a zero T so empty entries
// round-trip without erroring. resource and v name the blob in errors ("log data").
func DecodeBlob[T any](
	blob msgpack.EncodedJSON,
	resource string,
	v Version,
) (T, error) {
	var d T
	if blob == nil {
		return d, nil
	}
	if err := blob.Unmarshal(&d); err != nil {
		return d, errors.Wrapf(err, "decode v%d %s", v, resource)
	}
	return d, nil
}
