// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicertificate "github.com/synnaxlabs/synnax/pkg/api/certificate"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPICertificate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Certificate Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	svc    *service.Layer
	apiSvc *apicertificate.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	svc = MustOpen(svctestutil.OpenLayer(ctx, node))
	apiSvc = MustSucceed(apicertificate.NewService(apicertificate.ServiceConfig{
		Certificate: svc.Certificate,
		RBAC:        svc.RBAC,
	}))
})

// createUser creates a user whose role allows nothing.
func createUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(rbactestutil.CreateUser(ctx, svc.RBAC, svc.User, user.User{
		Name: "user-" + uuid.New().String(),
	}))
}

// grant gives the subject the actions on the objects.
func grant(
	ctx SpecContext,
	subject resource.ID,
	actions []access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	MustSucceed(rbactestutil.Grant(ctx, svc.RBAC, svc.User, key, actions, objects...))
}
