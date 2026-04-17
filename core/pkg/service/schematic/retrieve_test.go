// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Retrieve", func() {
	It("Should retrieve a Schematic", func(ctx SpecContext) {
		s := schematic.Schematic{
			Name: "test",
			Props: map[string]msgpack.EncodedJSON{
				"key": map[string]any{"data": "data_two"},
			}}
		Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
		var res schematic.Schematic
		Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
		Expect(res.Key).To(Equal(s.Key))
		Expect(res.Name).To(Equal(s.Name))
		Expect(res.Props).To(Equal(s.Props))
	})
})
