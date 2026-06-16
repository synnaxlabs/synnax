// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"fmt"
	"math/rand"
)

// NewRandomName generates a random channel name that should be unique. It lives in the
// distribution layer as a cross-layer test helper so distribution-layer tests can
// generate channel names without depending on the service layer, where channel name
// validation (ValidateName) lives.
func NewRandomName() string {
	randomSuffix := rand.Intn(999999999)
	return fmt.Sprintf("test_ch_%09d", randomSuffix)
}
