// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

var (
	kvDB kv.DB
	db   *gorp.DB
)
var _ = BeforeSuite(func() {
	kvDB = memkv.New()
	db = gorp.Wrap(kvDB)
})

var _ = AfterSuite(func() {
	Expect(db.Close()).To(Succeed())
})

func TestGorp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gorp Suite")
}
