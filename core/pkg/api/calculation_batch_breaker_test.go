// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"fmt"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation batch breaker", func() {
	virtual := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	It(
		"Should not stream an edit of a batch that names a channel the subject "+
			"cannot read when a later edit of the batch undoes it",
		func(ctx SpecContext) {
			visible, secret := virtual(ctx), virtual(ctx)
			c := calculation.Calculation{
				Expression: fmt.Sprintf("return %s * 2", visible.Name),
			}
			var output channel.Channel
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				created, err := svc.Calculation.NewWriter(tx).
					CreateOutput(ctx, &c, UniqueChannelName(), nil)
				if err == nil {
					output = created[0]
				}
				return err
			})).To(Succeed())
			var actions channel.Channel
			Expect(svc.Channel.NewRetrieve().
				Where(channel.MatchNames("sy_calculation_action_set")).
				Entry(&actions).
				Exec(ctx, db)).To(Succeed())
			viewer := MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				svc.RBAC,
				svc.User,
				user.User{Name: "viewer-" + uuid.New().String()},
			))
			MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				svc.RBAC,
				svc.User,
				viewer.Key,
				[]access.Action{access.ActionRetrieve},
				resource.ID{Type: resource.TypeCalculation},
				channel.ResourceID(visible.Key()),
				channel.ResourceID(output.Key()),
				framer.ResourceIDs(channel.Keys{actions.Key()})[0],
			))
			client, server := freightertestutil.NewStreams[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			](ctx)
			done := make(chan error, 1)
			go func() {
				done <- apiLayer.Framer.Stream(AuthedCtx(ctx, viewer), server)
			}()
			DeferCleanup(func() {
				Expect(client.CloseSend()).To(Succeed())
				Eventually(done).WithTimeout(5 * time.Second).Should(Receive())
			})
			Expect(client.Send(apiframer.StreamerRequest{
				Keys: channel.Keys{actions.Key()},
			})).To(Succeed())
			Expect(MustSucceed(client.Receive()).Frame.Empty()).To(BeTrue())
			editor := newUser(
				ctx,
				resource.TypeChannel,
				resource.TypeCalculation,
				resource.TypeFramer,
			)
			MustSucceed(execBatch(ctx, editor, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/channel/dispatch",
					"payload": {"key": %[1]d, "actions": [{"type": "set", "set": {
						"path": ["expression"], "value": "return %[2]s * 2"
					}}]}
				},
				{
					"endpoint": "/channel/dispatch",
					"payload": {"key": %[1]d, "actions": [{"type": "set", "set": {
						"path": ["expression"], "value": "return %[3]s * 3"
					}}]}
				}
			]}`, output.Key(), secret.Name, visible.Name)))
			res := MustSucceed(client.Receive())
			var received []string
			for _, series := range res.Frame.Get(actions.Key()).Series {
				for sample := range series.Samples() {
					received = append(received, string(sample))
				}
			}
			Expect(received).ToNot(BeEmpty())
			for _, s := range received {
				Expect(s).ToNot(ContainSubstring(secret.Name))
			}
		},
	)
})
