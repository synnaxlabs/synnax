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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
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
	u := MustSucceed(svc.User.NewWriter(nil).Create(ctx, user.User{
		Name: "user-" + uuid.New().String(),
	}))
	objects := make([]resource.ID, len(types))
	for i, t := range types {
		objects[i] = resource.ID{Type: t}
	}
	roleWriter := svc.RBAC.Role.NewWriter(nil, true)
	policyWriter := svc.RBAC.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Role:    r.Key,
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: access.AllActions,
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(svc.User.NewWriter(nil).AssignRole(ctx, r.Key, u.Key)).To(Succeed())
	return u
}

// execBatch decodes body as a batch and runs it as u in one transaction.
func execBatch(ctx context.Context, u user.User, body string) error {
	GinkgoHelper()
	var req batch.ExecRequest
	Expect(json.Codec.Decode(ctx, []byte(body), &req)).To(Succeed())
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := apiLayer.Batch.Exec(AuthedCtx(ctx, u), tx, req)
		return err
	})
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
			Expect(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
				{
					"endpoint": "/group/create",
					"payload": {"key": %[1]q, "name": "Batched", "scope": "project"}
				},
				{
					"endpoint": "/project/set-group",
					"payload": {"keys": [%[2]q], "group": %[1]q}
				}
			]}`, key, proj.Key))).To(Succeed())
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
		]}`, key, proj.Key, missing))).To(MatchError(
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
			Expect(execBatch(ctx, u, fmt.Sprintf(`{"requests": [
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
			]}`, key, l.Key))).To(Succeed())
			var stored status.Status[any]
			Expect(svc.Status.NewRetrieve[any]().
				Where(status.MatchKeys[any](key)).
				Entry(&stored).
				Exec(ctx, nil)).To(Succeed())
			Expect(stored.Labels).To(Equal([]label.Key{l.Key}))
		},
	)
})
