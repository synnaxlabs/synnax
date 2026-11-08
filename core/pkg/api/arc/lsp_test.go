// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const lspURI = "file:///api.arc"

// lspClient sends JSON-RPC messages to the Arc LSP of the API over an in-memory stream.
type lspClient struct {
	stream *freightertestutil.ClientStream[apiarc.LSPMessage, apiarc.LSPMessage]
	id     int
}

// openLSP serves the Arc LSP to subject and opens a document that holds source.
func openLSP(subject resource.ID, source string) *lspClient {
	GinkgoHelper()
	ctx, cancel := context.WithCancel(context.Background())
	stream, server := freightertestutil.NewStreams[
		apiarc.LSPMessage,
		apiarc.LSPMessage,
	](ctx)
	errs := make(chan error, 1)
	go func() { errs <- apiSvc.LSP(SubjectCtx(ctx, subject), server) }()
	DeferCleanup(func() {
		cancel()
		Eventually(errs).Should(Receive())
	})
	c := &lspClient{stream: stream}
	c.call("initialize", map[string]any{"clientInfo": map[string]any{"name": "test"}})
	c.send(
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "initialized",
			"params":  map[string]any{},
		},
	)
	c.send(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{
			"uri":        lspURI,
			"languageId": "arc",
			"version":    1,
			"text":       source,
		}},
	})
	return c
}

func (c *lspClient) send(msg map[string]any) {
	GinkgoHelper()
	content := MustSucceed(json.Marshal(msg))
	Expect(c.stream.Send(apiarc.LSPMessage{Content: string(content)})).To(Succeed())
}

// call sends a request and returns the response to it.
func (c *lspClient) call(method string, params any) map[string]any {
	GinkgoHelper()
	c.id++
	c.send(
		map[string]any{
			"jsonrpc": "2.0",
			"id":      c.id,
			"method":  method,
			"params":  params,
		},
	)
	for {
		msg := MustSucceed(c.stream.Receive())
		var res map[string]any
		Expect(json.Unmarshal([]byte(msg.Content), &res)).To(Succeed())
		if _, isRequest := res["method"]; !isRequest && res["id"] != nil {
			Expect(res["id"]).To(BeEquivalentTo(c.id))
			return res
		}
	}
}

// at returns the params of a request on the document at line and character.
func at(line, character int) map[string]any {
	return map[string]any{
		"textDocument": map[string]any{"uri": lspURI},
		"position":     map[string]any{"line": line, "character": character},
	}
}

var _ = Describe("LSP", func() {
	var ch channel.Channel
	BeforeEach(func(ctx SpecContext) {
		ch = channel.Channel{
			Name:     "api_lsp_ch",
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
		})
	})

	It("Should hide the channels that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "lsp-reader"},
		))
		const source = "func test() {\n\tx := api_lsp_ch\n}"
		Expect(openLSP(u.ResourceID(), source).call("textDocument/hover", at(1, 8))).
			To(HaveKeyWithValue("result", BeNil()))
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, ch.ResourceID())
		Expect(openLSP(u.ResourceID(), source).call("textDocument/hover", at(1, 8))).
			To(HaveKeyWithValue("result", HaveKeyWithValue(
				"contents",
				HaveKeyWithValue("value", ContainSubstring("api_lsp_ch")),
			)))
	})

	Describe("Rename", func() {
		var u user.User
		BeforeEach(func(ctx SpecContext) {
			u = MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "lsp-renamer"},
			))
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, ch.ResourceID())
		})
		rename := func() map[string]any {
			GinkgoHelper()
			params := at(1, 8)
			params["newName"] = "api_lsp_renamed"
			return openLSP(u.ResourceID(), "func test() {\n\tx := api_lsp_ch\n}").
				call("textDocument/rename", params)
		}
		names := func(ctx SpecContext) (string, string) {
			GinkgoHelper()
			var stored channel.Channel
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			return stored.Name, MustSucceed(
				node.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
			).Name
		}

		It("Should not rename a channel that the subject cannot update", func(
			ctx SpecContext,
		) {
			denied := access.Denied(
				u.ResourceID(),
				access.ActionUpdate,
				ch.ResourceID(),
			)
			Expect(rename()).To(HaveKeyWithValue(
				"error",
				HaveKeyWithValue("message", ContainSubstring(denied.Error())),
			))
			gorpName, cesiumName := names(ctx)
			Expect(gorpName).To(Equal("api_lsp_ch"))
			Expect(cesiumName).To(Equal("api_lsp_ch"))
		})

		It(
			"Should rename a channel in both stores when the subject can update it",
			func(
				ctx SpecContext,
			) {
				grantOn(ctx, u.ResourceID(), access.ActionUpdate, ch.ResourceID())
				Expect(rename()).ToNot(HaveKey("error"))
				gorpName, cesiumName := names(ctx)
				Expect(gorpName).To(Equal("api_lsp_renamed"))
				Expect(cesiumName).To(Equal("api_lsp_renamed"))
			},
		)
	})
})
