// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package connectivity_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	svcmock "github.com/synnaxlabs/synnax/pkg/service/mock"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check", func() {
	var (
		db     kv.DB
		signer svcmock.Signer
	)
	BeforeEach(func() {
		db = DeferClose(memkv.New())
		signer = svcmock.NewSigner()
	})
	openLicense := func(
		ctx SpecContext,
		cfgs ...license.ServiceConfig,
	) *license.Service {
		GinkgoHelper()
		return MustOpen(license.OpenService(ctx, append(
			[]license.ServiceConfig{{DB: db, Anchors: signer.Anchors}},
			cfgs...,
		)...))
	}
	check := func(ctx SpecContext, lic *license.Service) connectivity.CheckResponse {
		GinkgoHelper()
		svc := MustSucceed(connectivity.NewService(apicfg.LayerConfig{
			Distribution: node.Layer,
			Service:      &service.Layer{License: lic},
		}))
		return MustSucceed(svc.Check(ctx, struct{}{}))
	}

	It("Should describe the Core", func(ctx SpecContext) {
		res := check(ctx, openLicense(ctx))
		Expect(res.ClusterKey).To(Equal(node.Cluster.Key().String()))
		Expect(res.NodeKey).To(Equal(node.Cluster.HostKey()))
		Expect(res.NodeVersion).To(Equal(version.Get()))
	})

	It("Should report a missing license", func(ctx SpecContext) {
		Expect(check(ctx, openLicense(ctx)).License).To(Equal(license.StateMissing))
	})

	It("Should report an active license", func(ctx SpecContext) {
		lic := openLicense(ctx, license.ServiceConfig{
			Key: signer.Sign(svcmock.NewLicense()),
		})
		Expect(check(ctx, lic).License).To(Equal(license.StateOk))
	})

	It("Should report an expired license", func(ctx SpecContext) {
		lic := openLicense(ctx, license.ServiceConfig{
			Key: signer.Sign(svcmock.NewLicense()),
		})
		Expect(lic.Close()).To(Succeed())
		later := time.Now().AddDate(60, 0, 0)
		lic = openLicense(ctx, license.ServiceConfig{
			Now: func() time.Time { return later },
		})
		Expect(check(ctx, lic).License).To(Equal(license.StateExpired))
	})
})
