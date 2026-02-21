// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
)

var _ = Describe("Retrieve", func() {
	It("Should retrieve a LinePlot", func() {
		p := lineplot.LinePlot{Name: "test", Data: map[string]any{"key": "data"}}
		Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &p)).To(Succeed())
		var res lineplot.LinePlot
		Expect(svc.NewRetrieve().WhereKeys(p.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
		Expect(res).To(Equal(p))
	})
})
