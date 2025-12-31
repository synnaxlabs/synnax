// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/jerky/state"
)

var _ = Describe("State", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "jerky-state-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("File operations", func() {
		Context("Load", func() {
			It("should return new file when state doesn't exist", func() {
				f, err := state.Load(tempDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(f).ToNot(BeNil())
				Expect(f.Schema).To(Equal(state.SchemaVersion))
				Expect(f.Types).To(BeEmpty())
			})

			It("should load existing state file", func() {
				// Create and save a file first
				f := state.NewFile()
				f.SetTypeState("User", state.TypeState{
					Package:        "example",
					CurrentVersion: 2,
				})
				err := f.Save(tempDir)
				Expect(err).ToNot(HaveOccurred())

				// Load it back
				loaded, err := state.Load(tempDir)
				Expect(err).ToNot(HaveOccurred())
				ts, exists := loaded.GetTypeState("User")
				Expect(exists).To(BeTrue())
				Expect(ts.Package).To(Equal("example"))
				Expect(ts.CurrentVersion).To(Equal(2))
			})
		})

		Context("Save", func() {
			It("should create state file on disk", func() {
				f := state.NewFile()
				f.SetTypeState("Test", state.TypeState{
					Package: "test",
				})
				err := f.Save(tempDir)
				Expect(err).ToNot(HaveOccurred())

				path := filepath.Join(tempDir, state.StateFileName)
				_, err = os.Stat(path)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("TypeState", func() {
		Context("GetFieldNumber", func() {
			It("should assign sequential field numbers", func() {
				ts := state.TypeState{}
				num1 := ts.GetFieldNumber("Name")
				num2 := ts.GetFieldNumber("Age")
				num3 := ts.GetFieldNumber("Email")

				Expect(num1).To(Equal(1))
				Expect(num2).To(Equal(2))
				Expect(num3).To(Equal(3))
			})

			It("should return same number for same field", func() {
				ts := state.TypeState{}
				first := ts.GetFieldNumber("Name")
				second := ts.GetFieldNumber("Name")
				Expect(first).To(Equal(second))
			})

			It("should preserve field numbers across versions", func() {
				ts := state.TypeState{}
				_ = ts.GetFieldNumber("Name")  // 1
				_ = ts.GetFieldNumber("Age")   // 2
				_ = ts.GetFieldNumber("Email") // 3

				// Simulate field removal (Age is no longer in struct)
				// Then add a new field
				newNum := ts.GetFieldNumber("Score")
				Expect(newNum).To(Equal(4)) // Should be 4, not 2

				// Original fields keep their numbers
				Expect(ts.GetFieldNumber("Name")).To(Equal(1))
				Expect(ts.GetFieldNumber("Age")).To(Equal(2)) // Still 2 even if removed
			})
		})

		Context("AddVersion", func() {
			It("should add version to history", func() {
				ts := state.TypeState{}
				ts.AddVersion(state.VersionHistory{
					Version:       1,
					CreatedAt:     time.Now(),
					StructHash:    "abc123",
					CompositeHash: "def456",
				})

				Expect(ts.CurrentVersion).To(Equal(1))
				Expect(ts.History).To(HaveLen(1))
			})

			It("should track multiple versions", func() {
				ts := state.TypeState{}
				ts.AddVersion(state.VersionHistory{Version: 1})
				ts.AddVersion(state.VersionHistory{Version: 2})
				ts.AddVersion(state.VersionHistory{Version: 3})

				Expect(ts.CurrentVersion).To(Equal(3))
				Expect(ts.History).To(HaveLen(3))
			})
		})

		Context("LatestVersion", func() {
			It("should return nil for empty history", func() {
				ts := state.TypeState{}
				Expect(ts.LatestVersion()).To(BeNil())
			})

			It("should return latest version", func() {
				ts := state.TypeState{}
				ts.AddVersion(state.VersionHistory{Version: 1, StructHash: "first"})
				ts.AddVersion(state.VersionHistory{Version: 2, StructHash: "second"})

				latest := ts.LatestVersion()
				Expect(latest).ToNot(BeNil())
				Expect(latest.Version).To(Equal(2))
				Expect(latest.StructHash).To(Equal("second"))
			})
		})

		Context("GetVersion", func() {
			It("should return nil for non-existent version", func() {
				ts := state.TypeState{}
				Expect(ts.GetVersion(5)).To(BeNil())
			})

			It("should return specific version", func() {
				ts := state.TypeState{}
				ts.AddVersion(state.VersionHistory{Version: 1, StructHash: "v1"})
				ts.AddVersion(state.VersionHistory{Version: 2, StructHash: "v2"})
				ts.AddVersion(state.VersionHistory{Version: 3, StructHash: "v3"})

				v2 := ts.GetVersion(2)
				Expect(v2).ToNot(BeNil())
				Expect(v2.StructHash).To(Equal("v2"))
			})
		})
	})
})
