// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	"bytes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
)

type myReporter struct {
}

func (m myReporter) Report() alamos.Report {
	return alamos.Report{
		"key": "value",
	}
}

var _ = Describe("Report", func() {
	It("Should write the Instrumentation to JSON", func() {
		exp := alamos.New("exp")
		g := alamos.NewGauge[int](exp, alamos.Dev, "gauge")
		g.Record(1)
		g2 := alamos.NewGauge[int](exp, alamos.Dev, "gauge2")
		g2.Record(2)
		sub := alamos.Sub(exp, "sub")
		g3 := alamos.NewSeries[float64](sub, alamos.Dev, "gauge3")
		g3.Record(3.2)
		_ = alamos.NewSeries[float64](nil, alamos.Dev, "gauge4")
		w := bytes.NewBuffer([]byte{})
		err := exp.Report().WriteJSON(w)
		Expect(err).To(BeNil())
		Expect(w.String()).To(ContainSubstring("gauge"))
		Expect(exp.Report().String()).To(ContainSubstring("gauge"))
	})
	It("Should Attach reporters to an Instrumentation", func() {
		exp := alamos.New("exp")
		alamos.AttachReporter(exp, "reporter", alamos.Dev, myReporter{})
		Expect(exp.Report().String()).To(ContainSubstring("key"))
	})
})
