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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Calculated channel diagnostics check", func() {
	It("Should keep the diagnostics of an old API create that does not compile", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "diagnostics-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
		)
		_, err := createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return missing * 2",
			}},
		})
		Expect(err).To(MatchError(ContainSubstring("missing")))
		_, ok := errors.AsType[*diagnostics.Diagnostics](err)
		Expect(ok).To(BeTrue(), "%v holds diagnostics", err)
		pathErr := MustBeOk(errors.AsType[validate.PathError](err))
		Expect(pathErr.Path).To(HaveLen(2))
		Expect(pathErr.Path[1]).To(Equal("expression"))
	})
})
