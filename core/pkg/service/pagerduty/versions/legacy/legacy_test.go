// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Alert", func() {
	It("Should flip the per-alert enabled polarity", func() {
		in := msgpack.EncodedJSON{
			"routing_key": "rk",
			"alerts":      []any{map[string]any{"enabled": false}},
		}
		Expect(legacy.Alert.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"routing_key": "rk",
			"alerts":      []any{map[string]any{"disabled": true}},
		}))
	})
})
