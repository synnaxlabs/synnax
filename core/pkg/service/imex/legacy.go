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
	"strings"

	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// RequireFields errors when body omits any of fields, the structural markers of
// resource's Console state. A field may be a dotted path ("data.svg"). resource is a
// noun phrase carrying its own article ("an Arc", "a line plot").
//
// Import-only: the legacy chain decodes an unrecognized body to an empty resource, so a
// misrouted file would otherwise import as a blank one. The test is key presence, so a
// genuinely empty resource still passes.
func RequireFields(body msgpack.EncodedJSON, resource string, fields ...string) error {
	for _, f := range fields {
		if !hasPath(body, f) {
			return validate.PathedError(
				errors.Wrapf(
					validate.ErrValidation,
					"file is not %s: no %q field", resource, f,
				),
				f,
			)
		}
	}
	return nil
}

// hasPath reports whether body holds the dotted path, descending into a nested object
// at every segment but the last.
func hasPath(body map[string]any, path string) bool {
	for {
		key, rest, nested := strings.Cut(path, ".")
		v, ok := body[key]
		if !ok {
			return false
		}
		if !nested {
			return true
		}
		if body, ok = v.(map[string]any); !ok {
			return false
		}
		path = rest
	}
}

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
func (v Version) DecodeBlob[T any](
	blob msgpack.EncodedJSON,
	resource string,
) (T, error) {
	var t T
	if blob == nil {
		return t, nil
	}
	if err := blob.Unmarshal(&t); err != nil {
		return t, errors.Wrapf(err, "decode v%d %s", v, resource)
	}
	return t, nil
}
