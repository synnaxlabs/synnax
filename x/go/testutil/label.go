// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import "github.com/onsi/ginkgo/v2"

var (
	Integration = ginkgo.Label("integration")
	Performance = ginkgo.Label("performance")
	Slow        = ginkgo.Label("slow")
)
