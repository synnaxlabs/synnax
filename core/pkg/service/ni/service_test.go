// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ni_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *ni.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(ni.OpenService(ctx, ni.ServiceConfig{
			DB: db,
		}))
	})

	Describe("Stores", func() {
		It("Should expose one store per NI task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"ni_analog_read",
				"ni_analog_write",
				"ni_counter_read",
				"ni_digital_read",
				"ni_digital_write",
				"ni_scanner",
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded analog read config under the given key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.AnalogRead.Write(ctx, nil, key, msgpack.EncodedJSON{
				"sample_rate": 25,
				"stream_rate": 5,
				"channels":    []any{},
			})).To(Succeed())
			data := MustSucceed(svc.AnalogRead.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})
	})
})
