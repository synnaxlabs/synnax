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
	"context"
	"fmt"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// newUser creates a user that may take every action on resources of types.
func newUser(ctx context.Context, types ...resource.Type) user.User {
	GinkgoHelper()
	u := MustSucceed(rbactestutil.CreateUser(ctx, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
	objects := make([]resource.ID, len(types))
	for i, t := range types {
		objects[i] = resource.ID{Type: t}
	}
	MustSucceed(rbactestutil.Grant(
		ctx,
		svc.RBAC,
		svc.User,
		u.Key,
		access.AllActions,
		objects...,
	))
	return u
}

// execBatch decodes body as a batch and runs it as u in one transaction.
func execBatch(
	ctx context.Context,
	u user.User,
	body string,
) (batch.ExecResponse, error) {
	GinkgoHelper()
	var req batch.ExecRequest
	Expect(json.Codec.Decode(ctx, []byte(body), &req)).To(Succeed())
	var res batch.ExecResponse
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		var err error
		res, err = apiLayer.Batch.Exec(AuthedCtx(ctx, u), tx, req)
		return err
	}); err != nil {
		return batch.ExecResponse{}, err
	}
	return res, nil
}

var _ = Describe("Layer", func() {
	DescribeTable("Should refuse a batch request that writes channel storage",
		func(ctx SpecContext, endpoint string) {
			Expect(apiLayer.Batch.Exec(ctx, nil, batch.ExecRequest{
				Requests: []batch.Request{{Endpoint: endpoint}},
			})).Error().To(MatchError(ContainSubstring(fmt.Sprintf(
				"requests.0.endpoint: %q cannot run in a batch",
				endpoint,
			))))
		},
		Entry("channel create", "/channel/create"),
		Entry("channel delete", "/channel/delete"),
	)

	It(
		"Should create a group and move a project into it in one transaction",
		func(ctx SpecContext) {
			u := newUser(ctx, resource.TypeGroup, resource.TypeProject)
			proj := project.Project{Name: "batched"}
			Expect(svc.Project.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
			key := uuid.New()
			MustSucceed(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/group/create",
					"payload": {"key": %[1]q, "name": "Batched", "scope": "project"}
				},
				{
					"endpoint": "/project/set-group",
					"payload": {"keys": [%[2]q], "group": %[1]q}
				}
			]}`, key, proj.Key)))
			var stored project.Project
			Expect(svc.Project.NewRetrieve().
				Where(project.MatchKeys(proj.Key)).
				Entry(&stored).
				Exec(ctx, nil)).To(Succeed())
			Expect(stored.Group).To(Equal(&key))
		},
	)

	It("Should commit nothing when a later request fails", func(ctx SpecContext) {
		u := newUser(ctx, resource.TypeGroup, resource.TypeProject)
		proj := project.Project{Name: "unbatched"}
		Expect(svc.Project.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
		key, missing := uuid.New(), uuid.New()
		Expect(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
			{
				"endpoint": "/group/create",
				"payload": {"key": %[1]q, "name": "Discarded", "scope": "project"}
			},
			{
				"endpoint": "/project/set-group",
				"payload": {"keys": [%[2]q], "group": %[3]q}
			}
		]}`, key, proj.Key, missing))).Error().To(MatchError(
			"requests.1.group: group " + missing.String() + " does not exist: " +
				"validation error",
		))
		Expect(svc.Group.NewRetrieve().
			Where(group.MatchKeys(key)).
			Entry(&group.Group{}).
			Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
	})

	It(
		"Should set a status and its labels in one transaction",
		func(ctx SpecContext) {
			u := newUser(ctx, resource.TypeStatus, resource.TypeLabel)
			l := label.Label{Name: "batched"}
			Expect(svc.Label.NewWriter(nil).Create(ctx, &l)).To(Succeed())
			key := uuid.New().String()
			MustSucceed(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/status/set",
					"payload": {"statuses": [{
						"key": %[1]q,
						"name": "Batched",
						"variant": "info",
						"message": "Batched status",
						"time": 1
					}]}
				},
				{
					"endpoint": "/label/set",
					"payload": {
						"id": {"type": "status", "key": %[1]q},
						"labels": [%[2]q],
						"replace": true
					}
				}
			]}`, key, l.Key)))
			var stored status.Status[any]
			Expect(svc.Status.NewRetrieve[any]().
				Where(status.MatchKeys[any](key)).
				Entry(&stored).
				Exec(ctx, nil)).To(Succeed())
			Expect(stored.Labels).To(Equal([]label.Key{l.Key}))
		},
	)

	It(
		"Should stream only the certificate changes that the subject can retrieve",
		func(ctx SpecContext) {
			newCertificate := func(name string) certificate.Certificate {
				return certificate.Certificate{Variant: certificate.FileCertificate{
					Base:            certificate.Base{Key: uuid.New(), Name: name},
					CertificatePath: "/etc/" + name + ".pem",
				}}
			}
			denied, readable := newCertificate("denied"), newCertificate("readable")
			records := map[channel.Key]func([]byte) (resource.ID, error){
				svc.Certificate.SetChannelKey():    certificate.SetRecord,
				svc.Certificate.DeleteChannelKey(): certificate.DeleteRecord,
			}
			keys := channel.Keys{
				svc.Certificate.SetChannelKey(),
				svc.Certificate.DeleteChannelKey(),
			}
			u := MustSucceed(rbactestutil.CreateUser(ctx, svc.RBAC, svc.User, user.User{
				Name: "user-" + uuid.New().String(),
			}))
			MustSucceed(rbactestutil.Grant(
				ctx,
				svc.RBAC,
				svc.User,
				u.Key,
				[]access.Action{access.ActionRetrieve},
				append(svcframer.ResourceIDs(keys), readable.ResourceID())...,
			))
			client, server := freightertestutil.NewStreams[
				apiframer.StreamerRequest,
				apiframer.StreamerResponse,
			](ctx)
			done := make(chan error, 1)
			go func() { done <- apiLayer.Framer.Stream(AuthedCtx(ctx, u), server) }()
			DeferCleanup(func() {
				Expect(client.CloseSend()).To(Succeed())
				Eventually(done).WithTimeout(5 * time.Second).Should(Receive())
			})
			Expect(client.Send(apiframer.StreamerRequest{Keys: keys})).To(Succeed())
			Expect(MustSucceed(client.Receive()).Frame.Empty()).To(BeTrue())
			w := svc.Certificate.NewWriter(nil)
			Expect(w.Create(ctx, &denied, nil)).To(Succeed())
			Expect(w.Create(ctx, &readable, nil)).To(Succeed())
			Expect(w.Delete(ctx, denied.Base().Key)).To(Succeed())
			Expect(w.Delete(ctx, readable.Base().Key)).To(Succeed())
			for _, key := range keys {
				res := MustSucceed(client.Receive())
				Expect(res.Frame.KeysSlice()).To(ConsistOf(key))
				var received []resource.ID
				for sample := range res.Frame.SeriesAt(0).Samples() {
					received = append(received, MustSucceed(records[key](sample)))
				}
				Expect(received).To(Equal([]resource.ID{readable.ResourceID()}))
			}
		},
	)
})
