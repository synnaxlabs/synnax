// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package base64 provides utilities for encoding and decoding to Base64.
package base64

import (
	"encoding/base64"

	"github.com/samber/lo"
)

// MustDecode decodes the given Base64-encoded string and panics if it fails. MustDecode
// is intended for initialization of static variables.
func MustDecode(s string) string {
	msg := lo.Must(base64.StdEncoding.DecodeString(s))
	return string(msg)
}
