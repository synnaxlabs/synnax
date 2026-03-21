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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/backup"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
)

func buildArchive(manifest backup.Manifest, files map[string]any) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// Write manifest
	w, err := zw.Create("manifest.json")
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, json.NewEncoder(w).Encode(manifest)).To(Succeed())
	// Write other files
	for path, v := range files {
		w, err := zw.Create(path)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, json.NewEncoder(w).Encode(v)).To(Succeed())
	}
	ExpectWithOffset(1, zw.Close()).To(Succeed())
	return buf.Bytes()
}

var _ = Describe("Import", func() {
	Describe("Analyze", func() {
		It("Should classify same-key workspaces as identical", func() {
			ws := workspace.Workspace{
				Name:   "Analyze Identical WS",
				Layout: `{"key":"test"}`,
				Author: testAuthor.Key,
			}
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws)).To(Succeed())

			var buf bytes.Buffer
			Expect(svc.Export(ctx, backup.ExportRequest{
				WorkspaceKeys: []uuid.UUID{ws.Key},
			}, &buf)).To(Succeed())

			reader := bytes.NewReader(buf.Bytes())
			resp, err := svc.Analyze(ctx, reader, int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())

			var wsItems []backup.AnalysisItem
			for _, item := range resp.Items {
				if item.Type == "workspace" {
					wsItems = append(wsItems, item)
				}
			}
			Expect(wsItems).To(HaveLen(1))
			Expect(wsItems[0].Name).To(Equal("Analyze Identical WS"))
			Expect(wsItems[0].Status).To(Equal(backup.StatusIdentical))
		})

		It("Should classify workspaces without matches as new", func() {
			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"workspaces"},
			}, map[string]any{
				"workspaces/" + uuid.New().String() + ".json": backup.Workspace{
					Name:   "NonexistentWorkspace_" + uuid.New().String(),
					Key:    uuid.New(),
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{}`),
				},
			})
			reader := bytes.NewReader(archiveData)
			resp, err := svc.Analyze(ctx, reader, int64(len(archiveData)))
			Expect(err).ToNot(HaveOccurred())

			var wsItems []backup.AnalysisItem
			for _, item := range resp.Items {
				if item.Type == "workspace" {
					wsItems = append(wsItems, item)
				}
			}
			Expect(wsItems).To(HaveLen(1))
			Expect(wsItems[0].Status).To(Equal(backup.StatusNew))
		})
	})

	Describe("Import Workspaces", func() {
		It("Should import a new workspace", func() {
			uniqueName := "Import New WS " + uuid.New().String()
			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"workspaces"},
			}, map[string]any{
				"workspaces/" + uuid.New().String() + ".json": backup.Workspace{
					Name:   uniqueName,
					Key:    uuid.New(),
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{"imported":true}`),
				},
			})

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicySkip,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Imported).To(Equal(1))

			var found []workspace.Workspace
			Expect(svcLayer.Workspace.NewRetrieve().
				WhereNames(uniqueName).
				Entries(&found).
				Exec(ctx, nil)).To(Succeed())
			Expect(found).To(HaveLen(1))
			Expect(found[0].Name).To(Equal(uniqueName))
		})

		It("Should skip a conflicting workspace with skip policy", func() {
			ws := workspace.Workspace{
				Name:   "Import Skip WS " + uuid.New().String(),
				Layout: `{"original":true}`,
				Author: testAuthor.Key,
			}
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws)).To(Succeed())

			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"workspaces"},
			}, map[string]any{
				"workspaces/" + uuid.New().String() + ".json": backup.Workspace{
					Name:   ws.Name,
					Key:    uuid.New(),
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{"modified":true}`),
				},
			})

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicySkip,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Skipped).To(Equal(1))

			var found workspace.Workspace
			Expect(svcLayer.Workspace.NewRetrieve().
				WhereKeys(ws.Key).Entry(&found).Exec(ctx, nil)).To(Succeed())
			Expect(found.Layout).To(Equal(`{"original":true}`))
		})

		It("Should replace a conflicting workspace with replace policy", func() {
			ws := workspace.Workspace{
				Name:   "Import Replace WS " + uuid.New().String(),
				Layout: `{"original":true}`,
				Author: testAuthor.Key,
			}
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws)).To(Succeed())

			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"workspaces"},
			}, map[string]any{
				"workspaces/" + uuid.New().String() + ".json": backup.Workspace{
					Name:   ws.Name,
					Key:    uuid.New(),
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{"replaced":true}`),
				},
			})

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicyReplace,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Replaced).To(Equal(1))

			var found workspace.Workspace
			Expect(svcLayer.Workspace.NewRetrieve().
				WhereKeys(ws.Key).Entry(&found).Exec(ctx, nil)).To(Succeed())
			Expect(found.Layout).To(ContainSubstring("replaced"))
		})
	})

	Describe("Import Workspace Children", func() {
		It("Should import a new lineplot under an existing workspace", func() {
			ws := workspace.Workspace{
				Name:   "WS With Children " + uuid.New().String(),
				Layout: `{}`,
				Author: testAuthor.Key,
			}
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws)).To(Succeed())

			archiveWSKey := uuid.New()
			lpKey := uuid.New()
			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"workspaces"},
			}, map[string]any{
				"workspaces/" + archiveWSKey.String() + ".json": backup.Workspace{
					Name:   ws.Name,
					Key:    archiveWSKey,
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{}`),
				},
				"workspaces/" + archiveWSKey.String() + "/lineplots/" + lpKey.String() + ".json": backup.DataVisualization{
					Name: "Test LinePlot",
					Key:  lpKey,
					Data: json.RawMessage(`{"channels":{"x1":0,"y1":[]}}`),
				},
			})

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicySkip,
			})
			Expect(err).ToNot(HaveOccurred())
			// 1 workspace identical (existing, same layout) + 1 lineplot imported
			Expect(resp.Identical).To(Equal(1))
			Expect(resp.Imported).To(Equal(1))
		})

		It("Should import a new schematic under a new workspace", func() {
			archiveWSKey := uuid.New()
			schKey := uuid.New()
			wsName := "New WS With Schematic " + uuid.New().String()
			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"workspaces"},
			}, map[string]any{
				"workspaces/" + archiveWSKey.String() + ".json": backup.Workspace{
					Name:   wsName,
					Key:    archiveWSKey,
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{}`),
				},
				"workspaces/" + archiveWSKey.String() + "/schematics/" + schKey.String() + ".json": backup.Schematic{
					Name:     "Test Schematic",
					Key:      schKey,
					Data:     json.RawMessage(`{"nodes":[]}`),
					Snapshot: false,
				},
			})

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicyReplace,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Imported).To(Equal(2)) // workspace + schematic
		})
	})

	Describe("Import Users", func() {
		It("Should skip existing users", func() {
			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"users"},
			}, map[string]any{
				"users/" + testAuthor.Key.String() + ".json": testAuthor,
			})

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicySkip,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Identical).To(Equal(1))
		})
	})

	Describe("Per-Item Override", func() {
		It("Should respect per-item overrides over default policy", func() {
			ws1 := workspace.Workspace{
				Name:   "Override WS1 " + uuid.New().String(),
				Layout: `{"original":true}`,
				Author: testAuthor.Key,
			}
			ws2 := workspace.Workspace{
				Name:   "Override WS2 " + uuid.New().String(),
				Layout: `{"original":true}`,
				Author: testAuthor.Key,
			}
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws1)).To(Succeed())
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws2)).To(Succeed())

			archiveKey1 := uuid.New()
			archiveKey2 := uuid.New()
			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{"workspaces"},
			}, map[string]any{
				"workspaces/" + archiveKey1.String() + ".json": backup.Workspace{
					Name:   ws1.Name,
					Key:    archiveKey1,
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{"replaced":true}`),
				},
				"workspaces/" + archiveKey2.String() + ".json": backup.Workspace{
					Name:   ws2.Name,
					Key:    archiveKey2,
					Author: testAuthor.Key,
					Layout: json.RawMessage(`{"replaced":true}`),
				},
			})

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicySkip,
				Overrides: map[string]backup.ConflictPolicy{
					archiveKey1.String(): backup.PolicyReplace,
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Replaced).To(Equal(1))
			Expect(resp.Skipped).To(Equal(1))

			var found1 workspace.Workspace
			Expect(svcLayer.Workspace.NewRetrieve().
				WhereKeys(ws1.Key).Entry(&found1).Exec(ctx, nil)).To(Succeed())
			Expect(found1.Layout).To(ContainSubstring("replaced"))

			var found2 workspace.Workspace
			Expect(svcLayer.Workspace.NewRetrieve().
				WhereKeys(ws2.Key).Entry(&found2).Exec(ctx, nil)).To(Succeed())
			Expect(found2.Layout).To(ContainSubstring("original"))
		})
	})

	Describe("Round Trip", func() {
		It("Should export and reimport a workspace with children", func() {
			ws := workspace.Workspace{
				Name:   "Roundtrip WS " + uuid.New().String(),
				Layout: `{"key":"test"}`,
				Author: testAuthor.Key,
			}
			Expect(svcLayer.Workspace.NewWriter(nil).Create(ctx, &ws)).To(Succeed())

			lp := lineplot.LinePlot{Name: "RT LinePlot", Data: `{"channels":{"x1":0,"y1":[]}}`}
			Expect(svcLayer.LinePlot.NewWriter(nil).Create(ctx, ws.Key, &lp)).To(Succeed())

			sch := schematic.Schematic{Name: "RT Schematic", Data: `{"nodes":[]}`}
			Expect(svcLayer.Schematic.NewWriter(nil).Create(ctx, ws.Key, &sch)).To(Succeed())

			t := table.Table{Name: "RT Table", Data: `{"columns":[]}`}
			Expect(svcLayer.Table.NewWriter(nil).Create(ctx, ws.Key, &t)).To(Succeed())

			// Export
			var buf bytes.Buffer
			Expect(svc.Export(ctx, backup.ExportRequest{
				WorkspaceKeys: []uuid.UUID{ws.Key},
			}, &buf)).To(Succeed())

			// Delete originals
			Expect(svcLayer.LinePlot.NewWriter(nil).Delete(ctx, lp.Key)).To(Succeed())
			Expect(svcLayer.Schematic.NewWriter(nil).Delete(ctx, sch.Key)).To(Succeed())
			Expect(svcLayer.Table.NewWriter(nil).Delete(ctx, t.Key)).To(Succeed())
			Expect(svcLayer.Workspace.NewWriter(nil).Delete(ctx, ws.Key)).To(Succeed())

			// Analyze
			reader := bytes.NewReader(buf.Bytes())
			analyzeResp, err := svc.Analyze(ctx, reader, int64(buf.Len()))
			Expect(err).ToNot(HaveOccurred())
			for _, item := range analyzeResp.Items {
				if item.Type == "workspace" {
					Expect(item.Status).To(Equal(backup.StatusNew))
				}
			}

			// Import
			reader = bytes.NewReader(buf.Bytes())
			importResp, err := svc.Import(ctx, reader, int64(buf.Len()), backup.ImportRequest{
				DefaultPolicy: backup.PolicyReplace,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(importResp.Imported).To(BeNumerically(">=", 4))
			Expect(importResp.Errors).To(BeEmpty())
		})
	})

	Describe("Empty Import", func() {
		It("Should handle an archive with no sections", func() {
			archiveData := buildArchive(backup.Manifest{
				Version:  backup.Version,
				Sections: []string{},
			}, nil)

			reader := bytes.NewReader(archiveData)
			resp, err := svc.Import(ctx, reader, int64(len(archiveData)), backup.ImportRequest{
				DefaultPolicy: backup.PolicySkip,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Imported).To(Equal(0))
			Expect(resp.Skipped).To(Equal(0))
			Expect(resp.Replaced).To(Equal(0))
		})
	})

	Describe("PolicyFor", func() {
		It("Should return the override when set", func() {
			req := backup.ImportRequest{
				DefaultPolicy: backup.PolicySkip,
				Overrides: map[string]backup.ConflictPolicy{
					"key1": backup.PolicyReplace,
				},
			}
			Expect(req.PolicyFor("key1")).To(Equal(backup.PolicyReplace))
			Expect(req.PolicyFor("key2")).To(Equal(backup.PolicySkip))
		})
	})
})
