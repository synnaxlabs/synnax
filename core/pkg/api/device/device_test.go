// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apidevice "github.com/synnaxlabs/synnax/pkg/api/device"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func createDevice(ctx SpecContext, key string) device.Device {
	GinkgoHelper()
	d := device.Device{
		Key:      key,
		Rack:     svc.Rack.EmbeddedKey,
		Location: "dev1",
		Name:     "Before",
		Make:     "Test Make",
		Model:    "Test Model",
	}
	Expect(svc.Device.NewWriter(nil).Create(ctx, &d)).To(Succeed())
	return d
}

func retrieveDevice(ctx SpecContext, key string) device.Device {
	GinkgoHelper()
	var d device.Device
	Expect(svc.Device.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&d).
		Exec(ctx, nil)).To(Succeed())
	return d
}

var _ = Describe("Rename", func() {
	It("Should rename the device when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-granted")
		grant(ctx, access.ActionUpdate, u.OntologyID(), d.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: d.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveDevice(ctx, d.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Location).To(Equal(d.Location))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-empty")
		grant(ctx, access.ActionUpdate, u.OntologyID(), d.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: d.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing device", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionUpdate, u.OntologyID(), device.OntologyID("missing"))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: "missing", Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "rename-denied")
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apidevice.RenameRequest{Key: d.Key, Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Before"))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should attach only each device's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := createDevice(ctx, "retrieve-with-status")
		withoutStatus := createDevice(ctx, "retrieve-without-status")
		Expect(svc.Status.NewWriter(nil).Delete(
			ctx, withoutStatus.OntologyID().String(),
		)).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.OntologyID(),
			withStatus.OntologyID(), withoutStatus.OntologyID(),
		)
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{
				Where: &device.Where{Key: &filter.Set[device.Key]{
					In: []device.Key{withoutStatus.Key, withStatus.Key},
				}},
				IncludeStatus: true,
			},
		))
		Expect(res.Devices).To(HaveLen(2))
		Expect(res.Devices[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Devices[0].Status).To(BeNil())
		Expect(res.Devices[1].Key).To(Equal(withStatus.Key))
		Expect(res.Devices[1].Status.Key).To(Equal(withStatus.OntologyID().String()))
	})

	It("Should treat a missing key as a non-match", func(ctx SpecContext) {
		u := createUser(ctx)
		d := createDevice(ctx, "present_device")
		grant(ctx, access.ActionRetrieve, u.OntologyID(), d.OntologyID())
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, u),
			apidevice.RetrieveRequest{
				Where: &device.Where{Key: &filter.Set[device.Key]{
					In: []device.Key{d.Key, "missing_device"},
				}},
			},
		))
		Expect(res.Devices).To(HaveLen(1))
		Expect(res.Devices[0].Key).To(Equal(d.Key))
	})

	It("Should return a path error for an invalid regex", func(ctx SpecContext) {
		u := createUser(ctx)
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apidevice.RetrieveRequest{
			Where: &device.Where{Make: &filter.String{Regex: new("(a")}},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"where.make.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})

var _ = Describe("RetrieveRequest", func() {
	It("Should reject a request member that it does not declare", func() {
		var req apidevice.RetrieveRequest
		Expect(json.Unmarshal([]byte(`{"keys":["dev1"]}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "keys"`)),
		))
	})
})
