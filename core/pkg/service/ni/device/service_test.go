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
	"context"
	"iter"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		otg        *ontology.Ontology
		stat       *status.Service
		rackSvc    *rack.Service
		svc        *device.Service
		referenced set.Set[device.Key]
		tx         gorp.Tx
		rackKey    rack.Key
	)
	BeforeEach(func(ctx SpecContext) {
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
		}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    groupSvc,
			Label:    labelSvc,
		}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			HostProvider: testutil.NewStaticHostProvider(1),
			Status:       stat,
			DriverRole:   MustSucceed(racktestutil.CreateDriverRole(ctx, otg)),
		}))
		rackKey = rackSvc.EmbeddedKey
		referenced = set.New[device.Key]()
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Status:   stat,
			Referenced: func(
				_ context.Context,
				_ gorp.Tx,
				key device.Key,
			) (bool, error) {
				return referenced.Contains(key), nil
			},
		}))
		tx = DeferClose(db.OpenTx())
	})

	create := func(ctx context.Context) device.Device {
		GinkgoHelper()
		d := device.Device{
			Rack:   rackKey,
			Name:   "PXI-6255",
			Serial: uuid.New().String(),
			Model:  "PXI-6255",
			Alias:  "Mod1",
		}
		Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
		return d
	}
	retrieve := func(ctx context.Context, key device.Key) device.Device {
		GinkgoHelper()
		var d device.Device
		Expect(svc.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&d).
			Exec(ctx, tx)).To(Succeed())
		return d
	}
	retrieveStatus := func(ctx context.Context, key device.Key) device.Status {
		GinkgoHelper()
		var s device.Status
		Expect(gorp.NewRetrieve[status.Key, device.Status]().
			Where(gorp.MatchKeys[status.Key, device.Status](
				device.OntologyID(key).String(),
			)).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}
	parents := func(ctx context.Context, key device.Key) []ontology.ID {
		GinkgoHelper()
		var res []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(device.OntologyID(key)).
			TraverseTo(ontology.ParentsTraverser).
			Entries(&res).
			Exec(ctx, tx)).To(Succeed())
		return ontology.ResourceIDs(res)
	}

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*device.ServiceConfig)) {
				cfg := device.ServiceConfig{DB: db, Ontology: otg, Status: stat}
				unset(&cfg)
				Expect(device.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *device.ServiceConfig) { c.DB = nil }),
			Entry("ontology", "ontology", func(c *device.ServiceConfig) {
				c.Ontology = nil
			}),
			Entry("status", "status", func(c *device.ServiceConfig) { c.Status = nil }),
		)
	})

	Describe("Create", func() {
		It("Should mint a key and store the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(d.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, d.Key).Serial).To(Equal(d.Serial))
		})

		It("Should keep a key the caller sets", func(ctx SpecContext) {
			key := uuid.New()
			d := device.Device{Key: key, Rack: rackKey, Name: "Module"}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(d.Key).To(Equal(key))
		})

		It("Should reject a device without a name", func(ctx SpecContext) {
			d := device.Device{Rack: rackKey}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).
				To(MatchError(ContainSubstring("name: required")))
		})

		It("Should write an unknown status", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			Expect(s.Name).To(Equal(d.Name))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(device.StatusDetails{Rack: rackKey, Device: d.Key}))
		})

		It(
			"Should move an existing status to a replaced device",
			func(ctx SpecContext) {
				d := create(ctx)
				s := retrieveStatus(ctx, d.Key)
				s.Variant = status.VariantSuccess
				Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
				other := rack.Rack{Name: "Other"}
				Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
				d.Rack = other.Key
				d.Name = "Replaced"
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				got := retrieveStatus(ctx, d.Key)
				Expect(got.Name).To(Equal("Replaced"))
				Expect(got.Details.Rack).To(Equal(other.Key))
				Expect(got.Variant).To(Equal(status.VariantSuccess))
			},
		)

		It("Should keep a status that already exists", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(retrieveStatus(ctx, d.Key).Variant).To(Equal(status.VariantSuccess))
		})

		It("Should parent a standalone device to its rack", func(ctx SpecContext) {
			d := create(ctx)
			Expect(parents(ctx, d.Key)).To(ConsistOf(rackKey.OntologyID()))
		})
	})

	Describe("Update", func() {
		It("Should change one field and keep the others", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Identifier = "dev1"
					return d, nil
				},
			)).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Identifier).To(Equal("dev1"))
			Expect(got.Serial).To(Equal(d.Serial))
		})

		It("Should keep the key when change clears it", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Key = uuid.Nil()
					return d, nil
				},
			)).To(Succeed())
			Expect(retrieve(ctx, d.Key).Key).To(Equal(d.Key))
		})

		It("Should rename the status with the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Name = "Renamed"
					return d, nil
				},
			)).To(Succeed())
			Expect(retrieveStatus(ctx, d.Key).Name).To(Equal("Renamed"))
		})

		It("Should move the parent edge with the rack", func(ctx SpecContext) {
			d := create(ctx)
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Rack = other.Key
					return d, nil
				},
			)).To(Succeed())
			Expect(parents(ctx, d.Key)).To(ConsistOf(other.OntologyID()))
		})

		It("Should move the parent edge when change edits the chassis in place", func(
			ctx SpecContext,
		) {
			first, second := uuid.New(), uuid.New()
			chassisID := func(key uuid.UUID) ontology.ID {
				return ontology.ID{
					Type: ontology.ResourceTypeNiChassis,
					Key:  key.String(),
				}
			}
			Expect(otg.NewWriter(tx).DefineResources(
				ctx,
				chassisID(first),
				chassisID(second),
			)).To(Succeed())
			d := device.Device{Rack: rackKey, Name: "Module", Chassis: &first}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					*d.Chassis = second
					return d, nil
				},
			)).To(Succeed())
			var res []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(d.OntologyID()).
				TraverseTo(ontology.ParentsTraverser).
				ExcludeFieldData(true).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(ontology.ResourceIDs(res)).To(ConsistOf(chassisID(second)))
		})

		It("Should move the status with the rack", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			s.Message = "Connected"
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Rack = other.Key
					return d, nil
				},
			)).To(Succeed())
			got := retrieveStatus(ctx, d.Key)
			Expect(got.Details.Rack).To(Equal(other.Key))
			Expect(got.Variant).To(Equal(status.VariantSuccess))
			Expect(got.Message).To(Equal("Connected"))
		})

		It("Should keep the row when its status write fails without a tx", func(
			ctx SpecContext,
		) {
			d := device.Device{Rack: rackKey, Name: "Module"}
			Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, d.Key)).To(Succeed())
			})
			Expect(stat.NewWriter(nil).Delete(
				ctx,
				device.OntologyID(d.Key).String(),
			)).To(Succeed())
			Expect(svc.NewWriter(nil).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Name = "Renamed"
					return d, nil
				},
			)).To(MatchError(query.ErrNotFound))
			var got device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Entry(&got).
				Exec(ctx, nil)).To(Succeed())
			Expect(got.Name).To(Equal("Module"))
		})

		It("Should return the error change returns", func(ctx SpecContext) {
			d := create(ctx)
			err := errors.New("change failed")
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(device.Device) (device.Device, error) {
					return device.Device{}, err
				},
			)).To(MatchError(err))
		})

		It("Should reject an invalid result", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Update(
				ctx,
				d.Key,
				func(d device.Device) (device.Device, error) {
					d.Name = ""
					return d, nil
				},
			)).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Update(
				ctx,
				uuid.New(),
				func(d device.Device) (device.Device, error) { return d, nil },
			)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the device, status, and resource", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, d.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(gorp.NewRetrieve[status.Key, device.Status]().
				Where(gorp.MatchKeys[status.Key, device.Status](
					device.OntologyID(d.Key).String(),
				)).
				Exists(ctx, tx)).To(BeFalse())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(device.OntologyID(d.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should refuse a device a task references", func(ctx SpecContext) {
			d := create(ctx)
			referenced.Add(d.Key)
			Expect(svc.NewWriter(tx).Delete(ctx, d.Key)).
				To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, d.Key).Key).To(Equal(d.Key))
		})

		It("Should succeed for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It("Should match devices by serial", func(ctx SpecContext) {
			d := create(ctx)
			create(ctx)
			var got []device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchSerials(d.Serial)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Key).To(Equal(d.Key))
		})

		It("Should match devices by chassis", func(ctx SpecContext) {
			key := uuid.New()
			d := device.Device{Rack: rackKey, Name: "Module", Chassis: &key}
			Expect(otg.NewWriter(tx).DefineResources(ctx, ontology.ID{
				Type: ontology.ResourceTypeNiChassis,
				Key:  key.String(),
			})).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			create(ctx)
			var got []device.Device
			Expect(svc.NewRetrieve().
				Where(device.MatchChassis(key)).
				Entries(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(1))
			Expect(got[0].Key).To(Equal(d.Key))
		})
	})

	Describe("Ontology", func() {
		It("Should report its resource type", func() {
			Expect(svc.Type()).To(Equal(ontology.ResourceTypeNiDevice))
		})

		It("Should return one ontology ID per key", func() {
			key := uuid.New()
			Expect(device.OntologyIDs([]device.Key{key})).
				To(ConsistOf(device.OntologyID(key)))
		})

		Describe("OnChange", func() {
			var (
				changes    []ontology.Change
				disconnect func()
			)
			BeforeEach(func() {
				changes = nil
				disconnect = svc.OnChange(func(
					_ context.Context,
					reader iter.Seq[ontology.Change],
				) {
					changes = append(changes, slices.Collect(reader)...)
				})
				DeferCleanup(func() { disconnect() })
			})

			It("Should translate a set and a delete", func(ctx SpecContext) {
				d := device.Device{Rack: rackKey, Name: "Module"}
				Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())
				Expect(svc.NewWriter(nil).Delete(ctx, d.Key)).To(Succeed())
				id := device.OntologyID(d.Key).String()
				Expect(changes).To(HaveLen(2))
				Expect(changes[0].Variant).To(Equal(change.VariantSet))
				Expect(changes[0].Key).To(Equal(id))
				Expect(changes[0].Value.Name).To(Equal(d.Name))
				Expect(changes[1].Variant).To(Equal(change.VariantDelete))
				Expect(changes[1].Key).To(Equal(id))
			})

			It("Should stop after disconnecting", func(ctx SpecContext) {
				disconnect()
				d := device.Device{Rack: rackKey, Name: "Module"}
				Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(svc.NewWriter(nil).Delete(ctx, d.Key)).To(Succeed())
				})
				Expect(changes).To(BeEmpty())
			})
		})

		It("Should retrieve a device as a resource", func(ctx SpecContext) {
			d := create(ctx)
			res := MustSucceed(svc.RetrieveResource(ctx, d.Key.String(), tx))
			Expect(res.ID).To(Equal(device.OntologyID(d.Key)))
			Expect(res.Name).To(Equal(d.Name))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.RetrieveResource(ctx, "not-a-uuid", tx)).Error().
				To(MatchError(ContainSubstring("uuid")))
		})

		It("Should find a device by its name", func(ctx SpecContext) {
			d := create(ctx)
			res := MustSucceed(svc.Search(ctx, tx, d.Name, 10))
			Expect(res.Keys).To(ContainElement(device.OntologyID(d.Key)))
		})
	})
})
