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
	"github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Read", func() {
	It("Should normalize the sub-index spelling", func() {
		in := msgpack.EncodedJSON{
			"channels": []any{map[string]any{"subindex": float64(1)}},
		}
		Expect(legacy.Read.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"channels": []any{map[string]any{"sub_index": float64(1)}},
		}))
	})
})

var _ = Describe("Write", func() {
	It("Should normalize the sub-index spelling", func() {
		in := msgpack.EncodedJSON{
			"channels": []any{map[string]any{"subindex": float64(1)}},
		}
		Expect(legacy.Write.Apply(in)).To(Equal(msgpack.EncodedJSON{
			"channels": []any{map[string]any{"sub_index": float64(1)}},
		}))
	})
})
