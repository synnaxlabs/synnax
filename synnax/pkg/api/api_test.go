// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"go.uber.org/zap"
)

var _ = Describe("Api", func() {
	Describe("New", func() {
		It("Should open a new API without panicking", func() {
			Expect(func() {
				api.New(api.Config{
					Logger:  zap.NewNop(),
					Storage: &storage.Store{},
				})
			}).ToNot(Panic())
		})
	})
})
