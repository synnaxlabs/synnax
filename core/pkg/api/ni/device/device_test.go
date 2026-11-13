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
	"bytes"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apidevice "github.com/synnaxlabs/synnax/pkg/api/ni/device"
	"github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	deviceType = resource.ID{Type: resource.TypeNiDevice}
	statusType = resource.ID{Type: resource.TypeStatus}
	allWrites  = []access.Action{
		access.ActionCreate,
		access.ActionUpdate,
		access.ActionDelete,
	}
)

var _ = Describe("Service", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })

	// grantWrites lets u write any NI device, its status, and the rack it sits on.
	grantWrites := func(ctx SpecContext, r rack.Key) {
		GinkgoHelper()
		grant(ctx, u, allWrites, deviceType, statusType)
		grant(ctx, u, []access.Action{access.ActionUpdate}, r.ResourceID())
	}
	grantRetrieve := func(ctx SpecContext) {
		GinkgoHelper()
		grant(ctx, u, []access.Action{access.ActionRetrieve}, deviceType)
	}
	createDevice := func(ctx SpecContext, overrides ...func(*device.Device)) device.Device {
		GinkgoHelper()
		d := device.Device{
			Rack:         svc.Rack.EmbeddedKey,
			Name:         "PXI-6255",
			Serial:       uuid.New().String(),
			ResourceName: uuid.New().String(),
			Model:        "PXI-6255",
		}
		for _, override := range overrides {
			override(&d)
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NI.Devices.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NI.Devices.NewWriter(tx).Delete(ctx, d.Key)
			})).To(Succeed())
		})
		return d
	}
	retrieveDevice := func(ctx SpecContext, key device.Key) device.Device {
		GinkgoHelper()
		var d device.Device
		Expect(svc.NI.Devices.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, db)).To(Succeed())
		return d
	}
	deleteDevice := func(ctx SpecContext, key device.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NI.Devices.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
	}

	Describe("NewService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(field string, unset func(*apidevice.ServiceConfig)) {
				cfg := apidevice.ServiceConfig{
					Device: svc.NI.Devices,
					Status: svc.Status,
				}
				unset(&cfg)
				Expect(apidevice.NewService(cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("device", "device", func(c *apidevice.ServiceConfig) {
				c.Device = nil
			}),
			Entry("status", "status", func(c *apidevice.ServiceConfig) {
				c.Status = nil
			}),
		)

		It("Should let a later config override an earlier one", func() {
			base := apidevice.ServiceConfig{Device: svc.NI.Devices}
			Expect(apidevice.NewService(base, apidevice.ServiceConfig{
				Status: svc.Status,
			})).ToNot(BeNil())
		})
	})

	Describe("Create", func() {
		It("Should mint a key when writes are granted", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{{
					Rack:         svc.Rack.EmbeddedKey,
					Name:         "Module",
					Serial:       uuid.New().String(),
					ResourceName: uuid.New().String(),
					Model:        "PXI-6255",
				}}},
			))
			Expect(res.Devices[0].Key).ToNot(Equal(uuid.Nil()))
			deleteDevice(ctx, res.Devices[0].Key)
		})

		It("Should not echo a status the client sent", func(ctx SpecContext) {
			grantWrites(ctx, svc.Rack.EmbeddedKey)
			res := MustSucceed(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{{
					Rack:         svc.Rack.EmbeddedKey,
					Name:         "Module",
					Serial:       uuid.New().String(),
					ResourceName: uuid.New().String(),
					Model:        "PXI-6255",
					Status:       &device.Status{Message: "Connected"},
				}}},
			))
			Expect(res.Devices[0].Status).To(BeNil())
			deleteDevice(ctx, res.Devices[0].Key)
		})

		It("Should reject the request when create is not granted", func(
			ctx SpecContext,
		) {
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{{
					Rack:         svc.Rack.EmbeddedKey,
					Name:         "Module",
					ResourceName: uuid.New().String(),
					Model:        "PXI-6255",
				}}},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should replace an existing device", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			d.Name = "Replaced"
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("Replaced"))
		})

		It("Should not replace a device without update access", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			d.Name = "Replaced"
			Expect(create(
				testutil.AuthedCtx(ctx, u),
				apidevice.CreateRequest{Devices: []device.Device{d}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveDevice(ctx, d.Key).Name).To(Equal("PXI-6255"))
		})
	})

	Describe("Retrieve", func() {
		It("Should reject a named device when retrieve is not granted", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Key: &filter.Set[device.Key]{Eq: &d.Key}},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})

		It("Should leave out the devices the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Serial: &filter.String{Eq: &d.Serial}},
				},
			))
			Expect(res.Devices).To(BeEmpty())
		})

		It("Should filter by serial and attach the status", func(ctx SpecContext) {
			d := createDevice(ctx)
			createDevice(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where:         &device.Where{Serial: &filter.String{Eq: &d.Serial}},
					IncludeStatus: true,
				},
			))
			Expect(res.Devices).To(HaveLen(1))
			Expect(res.Devices[0].Key).To(Equal(d.Key))
			Expect(res.Devices[0].Status.Message).To(Equal("Status unknown"))
		})

		DescribeTable(
			"Should filter by each request field",
			func(ctx SpecContext, filter func(*apidevice.RetrieveRequest, device.Device)) {
				d := createDevice(ctx)
				other := rack.Rack{Name: "Other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				createDevice(ctx, func(o *device.Device) {
					o.Rack = other.Key
					o.Name = "Other"
					o.Model = "Other"
				})
				grantRetrieve(ctx)
				var req apidevice.RetrieveRequest
				filter(&req, d)
				res := MustSucceed(retrieve(testutil.AuthedCtx(ctx, u), req))
				Expect(res.Devices).To(HaveLen(1))
				Expect(res.Devices[0].Key).To(Equal(d.Key))
			},
			Entry("names", func(r *apidevice.RetrieveRequest, d device.Device) {
				r.Where = &device.Where{Name: &filter.String{Eq: &d.Name}}
			}),
			Entry("models", func(r *apidevice.RetrieveRequest, d device.Device) {
				r.Where = &device.Where{Model: &filter.String{Eq: &d.Model}}
			}),
			Entry("racks", func(r *apidevice.RetrieveRequest, d device.Device) {
				r.Where = &device.Where{Rack: &filter.Set[rack.Key]{Eq: &d.Rack}}
			}),
			Entry("search term", func(r *apidevice.RetrieveRequest, d device.Device) {
				r.SearchTerm = d.Name
			}),
		)

		It("Should cap the results at the limit", func(ctx SpecContext) {
			d := createDevice(ctx)
			createDevice(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Name: &filter.String{Eq: &d.Name}},
					Limit: new(uint32(1)),
				},
			))
			Expect(res.Devices).To(HaveLen(1))
		})

		Describe("Order", func() {
			var (
				prefix string
				byName []device.Key
			)
			BeforeEach(func(ctx SpecContext) {
				prefix = uuid.New().String()
				named := func(name string) device.Key {
					GinkgoHelper()
					return createDevice(
						ctx,
						func(d *device.Device) { d.Name = prefix + name },
					).Key
				}
				beta := named("beta")
				upper := named("Alpha")
				gamma := named("Gamma")
				lower := named("alpha")
				tie := []device.Key{upper, lower}
				// Gorp orders equal names by the trailing bytes of the key.
				slices.SortFunc(tie, func(a, b device.Key) int {
					slices.Reverse(a[:])
					slices.Reverse(b[:])
					return bytes.Compare(a[:], b[:])
				})
				byName = append(tie, beta, gamma)
				grantRetrieve(ctx)
			})
			page := func(ctx SpecContext, order apidevice.Order) []device.Key {
				GinkgoHelper()
				var (
					keys  []device.Key
					after *apidevice.Cursor
				)
				for {
					res := MustSucceed(retrieve(
						testutil.AuthedCtx(ctx, u),
						apidevice.RetrieveRequest{
							Where: &device.Where{Name: &filter.String{
								In: []string{
									prefix + "beta",
									prefix + "Alpha",
									prefix + "Gamma",
									prefix + "alpha",
								},
							}},
							Order: &order,
							After: after,
							Limit: new(uint32(1)),
						},
					))
					if len(res.Devices) == 0 {
						return keys
					}
					last := res.Devices[len(res.Devices)-1]
					keys = append(keys, last.Key)
					after = &apidevice.Cursor{Key: last.Key, Name: &last.Name}
				}
			}

			It("Should page by name, ignoring case", func(ctx SpecContext) {
				order := apidevice.Order{Field: apidevice.OrderFieldName}
				Expect(page(ctx, order)).To(Equal(byName))
			})

			It("Should page by name in descending order", func(ctx SpecContext) {
				order := apidevice.Order{
					Field:      apidevice.OrderFieldName,
					Descending: true,
				}
				desc := slices.Clone(byName)
				slices.Reverse(desc)
				Expect(page(ctx, order)).To(Equal(desc))
			})

			DescribeTable("Should reject a request",
				func(ctx SpecContext, req apidevice.RetrieveRequest, message string) {
					Expect(
						retrieve(testutil.AuthedCtx(ctx, u), req),
					).Error().
						To(SatisfyAll(
							BeAssignableToTypeOf(validate.PathError{}),
							MatchError(message),
						))
				},
				Entry(
					"that resumes an order without the order's value",
					apidevice.RetrieveRequest{
						Order: &apidevice.Order{Field: apidevice.OrderFieldName},
						After: &apidevice.Cursor{Key: uuid.New()},
					},
					"after.name: is required to resume an order by name: validation error",
				),
				Entry(
					"that sorts a search",
					apidevice.RetrieveRequest{
						SearchTerm: "PXI",
						Order:      &apidevice.Order{Field: apidevice.OrderFieldName},
					},
					"order: a search sorts by relevance: validation error",
				),
				Entry(
					"that sorts by an unknown field",
					apidevice.RetrieveRequest{Order: &apidevice.Order{Field: "color"}},
					`order.field: "color" is not a field to sort by: validation error`,
				),
			)
		})

		It("Should page past the after key", func(ctx SpecContext) {
			a := createDevice(ctx)
			b := createDevice(ctx)
			grantRetrieve(ctx)
			authed := testutil.AuthedCtx(ctx, u)
			req := apidevice.RetrieveRequest{
				Where: &device.Where{
					Name: &filter.String{In: []string{a.Name, b.Name}},
				},
				Limit: new(uint32(1)),
			}
			first := MustSucceed(retrieve(authed, req))
			Expect(first.Devices).To(HaveLen(1))
			req.After = &apidevice.Cursor{Key: first.Devices[0].Key}
			second := MustSucceed(retrieve(authed, req))
			Expect(second.Devices).To(HaveLen(1))
			Expect(second.Devices[0].Key).ToNot(Equal(first.Devices[0].Key))
		})

		It("Should reject a search that resumes after a key", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantRetrieve(ctx)
			Expect(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					SearchTerm: d.Name,
					After:      &apidevice.Cursor{Key: d.Key},
				},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should return a path error for a malformed where", func(ctx SpecContext) {
			grantRetrieve(ctx)
			Expect(retrieve(testutil.AuthedCtx(ctx, u), apidevice.RetrieveRequest{
				Where: &device.Where{Model: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("where.model.regex: error parsing regexp")),
				MatchError(validate.ErrValidation),
			))
		})

		It("Should return no devices for a zero limit", func(ctx SpecContext) {
			createDevice(ctx)
			grantRetrieve(ctx)
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{Limit: new(uint32(0))},
			))
			Expect(res.Devices).To(BeEmpty())
		})

		It("Should return no devices for a missing key", func(ctx SpecContext) {
			grantRetrieve(ctx)
			missing := uuid.New()
			res := MustSucceed(retrieve(
				testutil.AuthedCtx(ctx, u),
				apidevice.RetrieveRequest{
					Where: &device.Where{Key: &filter.Set[device.Key]{Eq: &missing}},
				},
			))
			Expect(res.Devices).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		rename := func(key device.Key) apidevice.UpdateRequest {
			return apidevice.UpdateRequest{
				Key: key,
				Update: device.Update{Variant: device.ConfigurationUpdate{
					Configuration: device.Configuration{
						Name:       "Renamed",
						Identifier: "dev1",
					},
				}},
			}
		}

		It("Should write only the configuration group", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(d.Key))).
				Error().ToNot(HaveOccurred())
			got := retrieveDevice(ctx, d.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Serial).To(Equal(d.Serial))
		})

		It("Should write only the identity group", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apidevice.UpdateRequest{
					Key: d.Key,
					Update: device.Update{Variant: device.IdentityUpdate{
						Identity: device.Identity{
							Rack:         d.Rack,
							Serial:       "02B",
							ResourceName: d.ResourceName,
							Model:        d.Model,
						},
					}},
				},
			)).Error().ToNot(HaveOccurred())
			got := retrieveDevice(ctx, d.Key)
			Expect(got.Serial).To(Equal("02B"))
			Expect(got.Name).To(Equal(d.Name))
		})

		It("Should reject a request with no group", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			Expect(update(
				testutil.AuthedCtx(ctx, u),
				apidevice.UpdateRequest{Key: d.Key},
			)).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should reject the request when update is not granted", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			Expect(update(testutil.AuthedCtx(ctx, u), rename(d.Key))).
				Error().To(MatchError(access.ErrDenied))
			Expect(retrieveDevice(ctx, d.Key).Name).To(Equal(d.Name))
		})
	})

	Describe("Delete", func() {
		It("Should remove the device when delete is granted", func(ctx SpecContext) {
			d := createDevice(ctx)
			grantWrites(ctx, d.Rack)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(svc.NI.Devices.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, db)).To(BeFalse())
		})

		It("Should reject the request when delete is not granted", func(
			ctx SpecContext,
		) {
			d := createDevice(ctx)
			Expect(del(
				testutil.AuthedCtx(ctx, u),
				apidevice.DeleteRequest{Keys: []device.Key{d.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveDevice(ctx, d.Key).Key).To(Equal(d.Key))
		})
	})
})
