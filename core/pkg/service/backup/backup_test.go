// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/backup"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
)

var _ = Describe("Backup", func() {
	Describe("Workspace Export", func() {
		var ws workspace.Workspace

		BeforeEach(func() {
			ws = workspace.Workspace{
				Name:   "Test Workspace",
				Layout: `{"key":"test"}`,
				Author: testAuthor.Key,
			}
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws)).To(Succeed())
		})

		It("Should export a workspace with its metadata", func() {
			r := exportAndOpen(backup.ExportRequest{WorkspaceKeys: []uuid.UUID{ws.Key}})

			manifest := readJSON[backup.Manifest](r, "manifest.json")
			Expect(manifest.Version).To(Equal(backup.Version))
			Expect(manifest.Sections).To(ContainElement("workspaces"))

			exported := readJSON[backup.Workspace](r, "workspaces/"+ws.Key.String()+".json")
			Expect(exported.Name).To(Equal("Test Workspace"))
			Expect(exported.Key).To(Equal(ws.Key))
			Expect(json.Valid(exported.Layout)).To(BeTrue())
		})

		It("Should export child line plots", func() {
			lp := lineplot.LinePlot{Name: "Test LP", Data: `{"channels":[1,2,3]}`}
			Expect(svcLayer.LinePlot.NewWriter(nil).Create(ctx, ws.Key, &lp)).To(Succeed())

			r := exportAndOpen(backup.ExportRequest{WorkspaceKeys: []uuid.UUID{ws.Key}})
			exported := readJSON[backup.DataVisualization](r,
				"workspaces/"+ws.Key.String()+"/lineplots/"+lp.Key.String()+".json")
			Expect(exported.Name).To(Equal("Test LP"))
			Expect(json.Valid(exported.Data)).To(BeTrue())
		})

		It("Should export child schematics", func() {
			s := schematic.Schematic{Name: "Test Schematic", Data: `{"nodes":[]}`}
			Expect(svcLayer.Schematic.NewWriter(nil).Create(ctx, ws.Key, &s)).To(Succeed())

			r := exportAndOpen(backup.ExportRequest{WorkspaceKeys: []uuid.UUID{ws.Key}})
			exported := readJSON[backup.Schematic](r,
				"workspaces/"+ws.Key.String()+"/schematics/"+s.Key.String()+".json")
			Expect(exported.Name).To(Equal("Test Schematic"))
		})

		It("Should export child tables", func() {
			t := table.Table{Name: "Test Table", Data: `{"columns":["a"]}`}
			Expect(svcLayer.Table.NewWriter(nil).Create(ctx, ws.Key, &t)).To(Succeed())

			r := exportAndOpen(backup.ExportRequest{WorkspaceKeys: []uuid.UUID{ws.Key}})
			exported := readJSON[backup.DataVisualization](r,
				"workspaces/"+ws.Key.String()+"/tables/"+t.Key.String()+".json")
			Expect(exported.Name).To(Equal("Test Table"))
		})

		It("Should export child logs", func() {
			l := log.Log{Name: "Test Log", Data: `{"entries":[]}`}
			Expect(svcLayer.Log.NewWriter(nil).Create(ctx, ws.Key, &l)).To(Succeed())

			r := exportAndOpen(backup.ExportRequest{WorkspaceKeys: []uuid.UUID{ws.Key}})
			exported := readJSON[backup.DataVisualization](r,
				"workspaces/"+ws.Key.String()+"/logs/"+l.Key.String()+".json")
			Expect(exported.Name).To(Equal("Test Log"))
		})
	})

	Describe("User Export", func() {
		It("Should export users directly", func() {
			r := exportAndOpen(backup.ExportRequest{UserKeys: []uuid.UUID{testAuthor.Key}})

			manifest := readJSON[backup.Manifest](r, "manifest.json")
			Expect(manifest.Sections).To(ContainElement("users"))

			exported := readJSON[user.User](r, "users/"+testAuthor.Key.String()+".json")
			Expect(exported.Username).To(Equal("test_backup_user"))
		})
	})

	Describe("Error Handling", func() {
		It("Should return an error for a nonexistent workspace key", func() {
			var buf bytes.Buffer
			err := svc.Export(ctx, backup.ExportRequest{
				WorkspaceKeys: []uuid.UUID{uuid.New()},
			}, &buf)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Empty Export", func() {
		It("Should produce a valid archive with no keys", func() {
			r := exportAndOpen(backup.ExportRequest{})
			manifest := readJSON[backup.Manifest](r, "manifest.json")
			Expect(manifest.Sections).To(BeEmpty())
		})
	})
})

func exportAndOpen(req backup.ExportRequest) *zip.Reader {
	var buf bytes.Buffer
	ExpectWithOffset(1, svc.Export(ctx, req, &buf)).To(Succeed())
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return r
}

func readJSON[T any](r *zip.Reader, name string) T {
	var f *zip.File
	for _, candidate := range r.File {
		if candidate.Name == name {
			f = candidate
			break
		}
	}
	ExpectWithOffset(1, f).ToNot(BeNil(), "file not found in archive: %s", name)
	rc, err := f.Open()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	defer func(rc io.ReadCloser) { _ = rc.Close() }(rc)
	var v T
	ExpectWithOffset(1, json.NewDecoder(rc).Decode(&v)).To(Succeed())
	return v
}
