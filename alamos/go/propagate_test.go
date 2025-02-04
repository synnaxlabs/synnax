// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
)

type mockCarrier struct {
	data map[string]string
}

var _ alamos.TraceCarrier = mockCarrier{}

func (m mockCarrier) Set(key, value string) {
	m.data[key] = value
}

func (m mockCarrier) Get(key string) string {
	return m.data[key]
}

func (m mockCarrier) Keys() []string {
	return lo.Keys(m.data)
}

var _ = Describe("Propagate", func() {
})
