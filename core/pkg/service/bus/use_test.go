// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bus_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/bus"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Uses", func() {
	It("Should return the messages and fields a task config uses", func() {
		message, field := uuid.New(), uuid.New()
		config := msgpack.EncodedJSON{
			"icd": uuid.New().String(),
			"messages": []any{
				map[string]any{
					"message": message.String(),
					"fields": []any{
						map[string]any{"field": field.String(), "channel": 3},
					},
				},
			},
		}
		Expect(MustSucceed(bus.Uses(config))).To(Equal([]icd.Use{
			{Message: message, Fields: []icd.FieldKey{field}},
		}))
	})

	It("Should return no uses for a config with no messages", func() {
		Expect(MustSucceed(bus.Uses(msgpack.EncodedJSON{}))).To(BeEmpty())
	})

	It("Should return an error for a config that does not decode", func() {
		Expect(bus.Uses(msgpack.EncodedJSON{"messages": "none"})).Error().
			To(MatchError(ContainSubstring(
				`unmarshal JSON string into Go slice within "/messages"`,
			)))
	})
})
