// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var db *gorp.DB

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
})

var _ = AfterSuite(func() {
	Expect(db.Close()).To(Succeed())
})

func TestDevice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Device Suite")
}
