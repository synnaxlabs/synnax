// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides test helpers for working with channels.
package testutil

import (
	"fmt"
	"math/rand"
)

// RandomName generates a random channel name that should be unique. Channel name
// uniqueness itself is enforced by the channel service during creation.
func RandomName() string {
	return fmt.Sprintf("test_ch_%09d", rand.Intn(999999999))
}
