// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv_test

import (
	"os"
	"path/filepath"

	pebblev1 "github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/kv/pebblekv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OpenReadOnly", func() {
	var (
		tempDir string
		dbPath  string
	)

	BeforeEach(func() {
		tempDir = MustSucceed(os.MkdirTemp("", "pebblekv-readonly-test-*"))
		dbPath = filepath.Join(tempDir, "db")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	createStore := func() {
		GinkgoHelper()
		db := MustSucceed(pebble.Open(dbPath, &pebble.Options{
			FS:                 vfs.Default,
			FormatMajorVersion: pebble.FormatNewest,
			Logger:             pebblekv.NewNoopLogger(),
		}))
		Expect(db.Set([]byte("dog"), []byte("woof"), pebble.Sync)).To(Succeed())
		Expect(db.Close()).To(Succeed())
	}

	It("Should read existing data and refuse writes", func(ctx SpecContext) {
		createStore()
		db := MustSucceed(pebblekv.OpenReadOnly(dbPath, alamos.Instrumentation{}))
		defer func() { Expect(db.Close()).To(Succeed()) }()
		v, closer := MustSucceed2(db.Get(ctx, []byte("dog")))
		Expect(v).To(Equal([]byte("woof")))
		Expect(closer.Close()).To(Succeed())
		Expect(db.Set(ctx, []byte("cat"), []byte("meow"))).To(
			MatchError(ContainSubstring("read-only")),
		)
	})

	It("Should hold the directory lock while open", func() {
		createStore()
		db := MustSucceed(pebblekv.OpenReadOnly(dbPath, alamos.Instrumentation{}))
		defer func() { Expect(db.Close()).To(Succeed()) }()
		Expect(pebble.Open(dbPath, &pebble.Options{
			FS:     vfs.Default,
			Logger: pebblekv.NewNoopLogger(),
		})).Error().To(MatchError(ContainSubstring("lock")))
	})

	It("Should refuse a store whose format requires migration", func() {
		oldDB := MustSucceed(pebblev1.Open(dbPath, &pebblev1.Options{
			FormatMajorVersion: pebblev1.FormatDefault,
			Logger:             pebblekv.NewNoopLogger(),
		}))
		Expect(oldDB.Close()).To(Succeed())
		Expect(pebblekv.OpenReadOnly(dbPath, alamos.Instrumentation{})).Error().To(
			MatchError(pebblekv.ErrFormatUnsupported),
		)
	})
})
