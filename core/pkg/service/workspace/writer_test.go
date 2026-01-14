// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a workspace", func() {
			ws := workspace.Workspace{
				Name:   "test",
				Author: author.Key,
				Layout: map[string]any{"key": "data"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(ws.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, ws.Key, "test2")).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetLayout", func() {
		It("Should set the layout of a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).SetLayout(ctx, ws.Key, map[string]any{"key": "data"})).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Layout["key"]).To(Equal("data"))
		})
	})
	Describe("DeleteChannel", func() {
		It("Should delete a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
		})
	})
})
