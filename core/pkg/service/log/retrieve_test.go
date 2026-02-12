// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/log"
)

var _ = Describe("Retrieve", func() {
	It("Should retrieve a Log", func() {
		l := log.Log{Name: "test", Data: "data"}
		Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
		var res log.Log
		Expect(svc.NewRetrieve().WhereKeys(l.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
		Expect(res).To(Equal(l))
	})
})
