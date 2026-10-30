// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Channel Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	channelSvc *channel.Service
	statusSvc  *status.Service
	rbacSvc    *rbac.Service
	apiSvc     *apichannel.Service
	author     user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
	}))
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		User:     userSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Group:    groupSvc,
		Ontology: otg,
		Label:    labelSvc,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
	}))
	rangerSvc := MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	aliasSvc := MustOpen(alias.OpenService(ctx, alias.ServiceConfig{
		DB:              node.DB,
		Ontology:        otg,
		Channel:         channelSvc,
		ParentRetriever: rangerSvc,
	}))
	apiSvc = MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
		RBAC:    rbacSvc,
		Channel: channelSvc,
		Ranger:  rangerSvc,
		Alias:   aliasSvc,
		Status:  statusSvc,
	}))
	author = MustSucceed(
		userSvc.NewWriter(nil).Create(ctx, user.User{Name: "test"}),
	)
})

// grantOn assigns the subject a role that allows the actions on the objects.
func grantOn(
	ctx SpecContext,
	subject ontology.ID,
	actions []access.Action,
	objects ...ontology.ID,
) {
	GinkgoHelper()
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	r := &role.Role{Name: "role-" + uuid.New().String(), Description: "test"}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    "policy-" + uuid.New().String(),
		Objects: objects,
		Actions: actions,
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, subject, r.Key)).To(Succeed())
}
