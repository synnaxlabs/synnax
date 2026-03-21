// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package export_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/export"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
)

var _ = Describe("Export", func() {
	var ws workspace.Workspace

	BeforeEach(func() {
		ws = workspace.Workspace{
			Name:   "Test Workspace",
			Layout: `{"key":"test"}`,
			Author: testAuthor.Key,
		}
		Expect(workspaceSvc.NewWriter(nil).Create(ctx, &ws)).To(Succeed())
	})

	Describe("Workspace Export", func() {
		It("Should export a workspace with its metadata", func() {
			var buf bytes.Buffer
			req := export.Request{WorkspaceKeys: []uuid.UUID{ws.Key}}
			Expect(svc.Export(ctx, req, &buf)).To(Succeed())

			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())

			manifestFile := findFile(r, "manifest.json")
			Expect(manifestFile).ToNot(BeNil())

			var manifest export.Manifest
			Expect(json.NewDecoder(
				mustOpen(manifestFile),
			).Decode(&manifest)).To(Succeed())
			Expect(manifest.Version).To(Equal(export.Version))
			Expect(manifest.Sections).To(ContainElement("workspaces"))

			wsFile := findFile(r, "workspaces/"+ws.Key.String()+".json")
			Expect(wsFile).ToNot(BeNil())

			var exportedWS export.Workspace
			Expect(json.NewDecoder(
				mustOpen(wsFile),
			).Decode(&exportedWS)).To(Succeed())
			Expect(exportedWS.Name).To(Equal("Test Workspace"))
			Expect(exportedWS.Key).To(Equal(ws.Key))
			Expect(exportedWS.Author).To(Equal(testAuthor.Key))
			Expect(json.Valid(exportedWS.Layout)).To(BeTrue())
		})

		It("Should export child line plots", func() {
			lp := lineplot.LinePlot{
				Name: "Test Line Plot",
				Data: `{"channels":[1,2,3]}`,
			}
			Expect(lineplotSvc.NewWriter(nil).Create(ctx, ws.Key, &lp)).To(Succeed())

			var buf bytes.Buffer
			req := export.Request{WorkspaceKeys: []uuid.UUID{ws.Key}}
			Expect(svc.Export(ctx, req, &buf)).To(Succeed())

			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())

			path := "workspaces/" + ws.Key.String() + "/lineplots/" + lp.Key.String() + ".json"
			lpFile := findFile(r, path)
			Expect(lpFile).ToNot(BeNil())

			var exportedLP export.LinePlot
			Expect(json.NewDecoder(
				mustOpen(lpFile),
			).Decode(&exportedLP)).To(Succeed())
			Expect(exportedLP.Name).To(Equal("Test Line Plot"))
			Expect(json.Valid(exportedLP.Data)).To(BeTrue())
		})

		It("Should export child schematics", func() {
			s := schematic.Schematic{
				Name: "Test Schematic",
				Data: `{"nodes":[]}`,
			}
			Expect(schematicSvc.NewWriter(nil).Create(ctx, ws.Key, &s)).To(Succeed())

			var buf bytes.Buffer
			req := export.Request{WorkspaceKeys: []uuid.UUID{ws.Key}}
			Expect(svc.Export(ctx, req, &buf)).To(Succeed())

			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())

			path := "workspaces/" + ws.Key.String() + "/schematics/" + s.Key.String() + ".json"
			sFile := findFile(r, path)
			Expect(sFile).ToNot(BeNil())

			var exportedS export.Schematic
			Expect(json.NewDecoder(
				mustOpen(sFile),
			).Decode(&exportedS)).To(Succeed())
			Expect(exportedS.Name).To(Equal("Test Schematic"))
			Expect(json.Valid(exportedS.Data)).To(BeTrue())
		})

		It("Should export child tables", func() {
			t := table.Table{
				Name: "Test Table",
				Data: `{"columns":["a","b"]}`,
			}
			Expect(tableSvc.NewWriter(nil).Create(ctx, ws.Key, &t)).To(Succeed())

			var buf bytes.Buffer
			req := export.Request{WorkspaceKeys: []uuid.UUID{ws.Key}}
			Expect(svc.Export(ctx, req, &buf)).To(Succeed())

			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())

			path := "workspaces/" + ws.Key.String() + "/tables/" + t.Key.String() + ".json"
			tFile := findFile(r, path)
			Expect(tFile).ToNot(BeNil())

			var exportedT export.Table
			Expect(json.NewDecoder(
				mustOpen(tFile),
			).Decode(&exportedT)).To(Succeed())
			Expect(exportedT.Name).To(Equal("Test Table"))
			Expect(json.Valid(exportedT.Data)).To(BeTrue())
		})

		It("Should export child logs", func() {
			l := log.Log{
				Name: "Test Log",
				Data: `{"entries":[]}`,
			}
			Expect(logSvc.NewWriter(nil).Create(ctx, ws.Key, &l)).To(Succeed())

			var buf bytes.Buffer
			req := export.Request{WorkspaceKeys: []uuid.UUID{ws.Key}}
			Expect(svc.Export(ctx, req, &buf)).To(Succeed())

			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())

			path := "workspaces/" + ws.Key.String() + "/logs/" + l.Key.String() + ".json"
			lFile := findFile(r, path)
			Expect(lFile).ToNot(BeNil())

			var exportedL export.Log
			Expect(json.NewDecoder(
				mustOpen(lFile),
			).Decode(&exportedL)).To(Succeed())
			Expect(exportedL.Name).To(Equal("Test Log"))
			Expect(json.Valid(exportedL.Data)).To(BeTrue())
		})

		It("Should produce a valid archive with no workspace keys", func() {
			var buf bytes.Buffer
			req := export.Request{}
			Expect(svc.Export(ctx, req, &buf)).To(Succeed())

			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())

			manifestFile := findFile(r, "manifest.json")
			Expect(manifestFile).ToNot(BeNil())

			var manifest export.Manifest
			Expect(json.NewDecoder(
				mustOpen(manifestFile),
			).Decode(&manifest)).To(Succeed())
			Expect(manifest.Sections).To(BeEmpty())
		})
	})
})

func findFile(r *zip.Reader, name string) *zip.File {
	for _, f := range r.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func mustOpen(f *zip.File) io.ReadCloser {
	rc, err := f.Open()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return rc
}
