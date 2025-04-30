// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crypto

import "encoding/base64"

// MustDecodeBase64 tries to decode the base64 encoded string. If decoding fails,
// MustDecodeBase64 will panic to simplify safe initialization of global variables.
func MustDecodeBase64(str string) string {
	msg, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return string(msg)
}
