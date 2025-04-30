// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package base64 provides extensions of the encoding/base64 package.
package base64

import "encoding/base64"

// MustDecode tries to decode the base64 encoded string. If decoding fails,
// MustDecode will panic to simplify safe initialization of global variables.
func MustDecode(str string) string {
	msg, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return string(msg)
}
