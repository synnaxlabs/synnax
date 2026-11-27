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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		stat       *status.Service
		rackSvc    *rack.Service
		svc        *device.Service
		channelSvc *channel.Service
		tx         gorp.Tx
		rackKey    rack.Key
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    node.Identity.HostKey,
			Status:     stat,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		}))
		rackKey = rackSvc.EmbeddedKey
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  stat,
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tx = DeferClose(db.OpenTx())
	})

	var serial uint32
	nextSerial := func() uint32 {
		serial++
		return serial
	}
	edit := func(d device.Device, change func(*device.Device)) device.Device {
		change(&d)
		return d
	}
	newModel := func(model device.Model, ports device.PortsVariant) device.Device {
		return device.Device{
			Rack:       rackKey,
			Name:       "LJ",
			Serial:     nextSerial(),
			Connection: device.ConnectionUSB,
			Model:      model,
			Ports:      device.Ports{Variant: ports},
		}
	}
	createChannels := func(ctx context.Context, count int) []channel.Key {
		GinkgoHelper()
		channels := make([]channel.Channel, count)
		for i := range channels {
			channels[i] = channel.Channel{
				Name:     channeltestutil.UniqueChannelName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
		}
		Expect(channelSvc.NewWriter(tx).CreateMany(ctx, &channels)).To(Succeed())
		return channel.KeysFromChannels(channels)
	}
	create := func(ctx context.Context) device.Device {
		GinkgoHelper()
		d := device.Device{
			Rack:       rackKey,
			Name:       "T7-0001",
			Serial:     nextSerial(),
			Connection: device.ConnectionUSB,
			Model:      device.ModelT7,
			Ports:      device.Ports{Variant: device.T7Ports{}},
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
		Expect(stat.NewRetrieve[device.StatusDetails]().
			Where(status.MatchOwners[device.StatusDetails](device.ResourceID(key))).
			Entry(&s).
			Exec(ctx, tx)).To(Succeed())
		return s
	}

	Describe("Channels", func() {
		createIndex := func(ctx context.Context) channel.Key {
			GinkgoHelper()
			ch := channel.Channel{
				Name:     channeltestutil.UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			Expect(channelSvc.NewWriter(tx).Create(ctx, &ch)).To(Succeed())
			return ch.Key()
		}
		bind := func(ctx context.Context, read *channel.Key) error {
			d := device.Device{
				Key:        uuid.New(),
				Rack:       rackKey,
				Name:       "T7-0002",
				Serial:     nextSerial(),
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
				Ports:      device.Ports{Variant: device.T7Ports{}},
				ReadIndex:  read,
			}
			return gorp.NewCreate[device.Key, device.Device]().Entry(&d).Exec(ctx, tx)
		}

		It("Should keep a channel that a device binds from being deleted", func(
			ctx SpecContext,
		) {
			ch := createIndex(ctx)
			Expect(bind(ctx, new(ch))).To(Succeed())
			Expect(channelSvc.NewWriter(tx).Delete(ctx, ch, false)).
				To(MatchError(validate.ErrValidation))
		})

		DescribeTable(
			"Should keep a channel that a port map holds from being deleted",
			func(ctx SpecContext, ports func(ch, other channel.Key) device.T7Ports) {
				channels := createChannels(ctx, 2)
				ch := channels[0]
				d := device.Device{
					Key:        uuid.New(),
					Rack:       rackKey,
					Name:       "T7-0003",
					Serial:     nextSerial(),
					Connection: device.ConnectionUSB,
					Model:      device.ModelT7,
					Ports:      device.Ports{Variant: ports(ch, channels[1])},
				}
				Expect(gorp.NewCreate[device.Key, device.Device]().
					Entry(&d).
					Exec(ctx, tx)).To(Succeed())
				Expect(channelSvc.NewWriter(tx).Delete(ctx, ch, false)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("an analog input", func(ch, _ channel.Key) device.T7Ports {
				return device.T7Ports{
					AnalogInput: map[device.T7AnalogInputPort]channel.Key{
						device.T7AnalogInputPortAin0: ch,
					},
				}
			}),
			Entry("a digital input", func(ch, _ channel.Key) device.T7Ports {
				return device.T7Ports{
					DigitalInput: map[device.T7DigitalPort]channel.Key{
						device.T7DigitalPortDio0: ch,
					},
				}
			}),
			Entry("an output command", func(ch, _ channel.Key) device.T7Ports {
				return device.T7Ports{
					AnalogOutput: map[device.AnalogOutputPort]device.CommandStatePair{
						device.AnalogOutputPortDac0: {Command: ch},
					},
				}
			}),
			Entry("an output state", func(ch, other channel.Key) device.T7Ports {
				return device.T7Ports{
					DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
						device.T7DigitalPortDio0: {Command: other, State: new(ch)},
					},
				}
			}),
		)
	})

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*device.ServiceConfig)) {
				cfg := device.ServiceConfig{DB: db, Status: stat}
				unset(&cfg)
				Expect(device.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *device.ServiceConfig) { c.DB = nil }),
			Entry("status", "status", func(c *device.ServiceConfig) { c.Status = nil }),
		)

		It("Should let a later config override an earlier one", func() {
			cfg := device.ServiceConfig{DB: db}.Override(
				device.ServiceConfig{Status: stat},
			)
			Expect(cfg.DB).To(BeIdenticalTo(db))
			Expect(cfg.Status).To(BeIdenticalTo(stat))
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Create", func() {
		It("Should mint a key and store the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(d.Key).ToNot(Equal(uuid.Nil()))
			Expect(retrieve(ctx, d.Key).Serial).To(Equal(d.Serial))
		})

		It("Should store the port maps of a device", func(ctx SpecContext) {
			ch := createChannels(ctx, 7)
			d := newModel(device.ModelT7, device.T7Ports{
				Mux80Attached: true,
				AnalogInput: map[device.T7AnalogInputPort]channel.Key{
					device.T7AnalogInputPortAin0:  ch[0],
					device.T7AnalogInputPortAin48: ch[1],
				},
				AnalogOutput: map[device.AnalogOutputPort]device.CommandStatePair{
					device.AnalogOutputPortDac1: {Command: ch[2], State: new(ch[3])},
				},
				DigitalInput: map[device.T7DigitalPort]channel.Key{
					device.T7DigitalPortDio0: ch[4],
				},
				DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
					device.T7DigitalPortDio19: {Command: ch[5], State: new(ch[6])},
				},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(retrieve(ctx, d.Key).Ports).To(Equal(d.Ports))
		})

		It("Should reject a port the model does not have", func(ctx SpecContext) {
			d := newModel(device.ModelT4, device.T4Ports{
				DigitalInput: map[device.T4DigitalPort]channel.Key{"DIO0": 5},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("invalid digital_input key: DIO0")),
			))
		})

		DescribeTable("Should allow a Mux80 port only with a Mux80 attached",
			func(ctx SpecContext, attached bool, matcher OmegaMatcher) {
				ch := createChannels(ctx, 2)
				d := newModel(device.ModelT7, device.T7Ports{
					Mux80Attached: attached,
					AnalogInput: map[device.T7AnalogInputPort]channel.Key{
						device.T7AnalogInputPortAin13: ch[0],
						device.T7AnalogInputPortAin48: ch[1],
					},
				})
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(matcher)
			},
			Entry("attached", true, Succeed()),
			Entry("not attached", false, SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"ports.analog_input: AIN48 needs mux80_attached",
				)),
			)),
		)

		DescribeTable("Should refuse a Mux80 address line with a Mux80 attached",
			func(ctx SpecContext, ports device.T7Ports, msg string) {
				ports.Mux80Attached = true
				d := newModel(device.ModelT7, ports)
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("digital input", device.T7Ports{
				DigitalInput: map[device.T7DigitalPort]channel.Key{
					device.T7DigitalPortDio20: 1,
				},
			}, "ports.digital_input: DIO20 addresses the attached Mux80"),
			Entry("digital output", device.T7Ports{
				DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
					device.T7DigitalPortDio22: {Command: 1},
				},
			}, "ports.digital_output: DIO22 addresses the attached Mux80"),
		)

		It("Should refuse to replace a device of another model", func(ctx SpecContext) {
			d := create(ctx)
			replaced := d
			replaced.Model = device.ModelT4
			replaced.Ports = device.Ports{Variant: device.T4Ports{}}
			Expect(svc.NewWriter(tx).Create(ctx, &replaced)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("model: cannot change from t7 to t4")),
			))
			Expect(retrieve(ctx, d.Key).Model).To(Equal(device.ModelT7))
		})

		It("Should keep a key the caller sets", func(ctx SpecContext) {
			key := uuid.New()
			d := device.Device{
				Key:        key,
				Rack:       rackKey,
				Name:       "T7-0001",
				Serial:     470010001,
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
				Ports:      device.Ports{Variant: device.T7Ports{}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(d.Key).To(Equal(key))
		})

		DescribeTable(
			"Should reject an invalid device",
			func(ctx SpecContext, change func(*device.Device), message string) {
				d := device.Device{
					Rack:       rackKey,
					Name:       "T7-0001",
					Serial:     470010001,
					Connection: device.ConnectionUSB,
					Model:      device.ModelT7,
					Ports:      device.Ports{Variant: device.T7Ports{}},
				}
				change(&d)
				Expect(svc.NewWriter(tx).Create(ctx, &d)).
					To(MatchError(ContainSubstring(message)))
			},
			Entry(
				"no name",
				func(d *device.Device) { d.Name = "" },
				"name: required",
			),
			Entry(
				"no serial",
				func(d *device.Device) { d.Serial = 0 },
				"serial: must be non-zero",
			),
			Entry(
				"no model",
				func(d *device.Device) { d.Model = "" },
				"model",
			),
			Entry(
				"no ports",
				func(d *device.Device) { d.Ports = device.Ports{} },
				"ports.model: required",
			),
			Entry(
				"ports for another model",
				func(d *device.Device) { d.Ports = device.Ports{Variant: device.T4Ports{}} },
				"ports: are for t4, not t7",
			),
			Entry(
				"unknown connection",
				func(d *device.Device) { d.Connection = "LJM_ctTCP" },
				"invalid connection",
			),
		)

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
				d = edit(d, func(b *device.Device) {
					b.Rack = other.Key
					b.Name = "Replaced"
				})
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				got := retrieveStatus(ctx, d.Key)
				Expect(got.Name).To(Equal("Replaced"))
				Expect(got.Details.Rack).To(Equal(other.Key))
				Expect(got.Variant).To(Equal(status.VariantSuccess))
			},
		)

		It("Should move the status when it moves the device to another rack", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			s.Message = "Connected"
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			moved := edit(d, func(b *device.Device) { b.Rack = other.Key })
			Expect(svc.NewWriter(tx).Create(ctx, &moved)).To(Succeed())
			got := retrieveStatus(ctx, d.Key)
			Expect(got.Details.Rack).To(Equal(other.Key))
			Expect(got.Variant).To(Equal(status.VariantSuccess))
			Expect(got.Message).To(Equal("Connected"))
		})

		It("Should keep a status that already exists", func(ctx SpecContext) {
			d := create(ctx)
			s := retrieveStatus(ctx, d.Key)
			s.Variant = status.VariantSuccess
			Expect(stat.NewWriter(tx).Set(ctx, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(
				retrieveStatus(ctx, d.Key).Variant,
			).To(Equal(status.VariantSuccess))
		})

		It("Should refuse a device on a rack that does not exist", func(
			ctx SpecContext,
		) {
			missing := rack.NewKey(1, 999)
			d := device.Device{
				Rack:       missing,
				Name:       "T7-0001",
				Serial:     470010001,
				Connection: device.ConnectionUSB,
				Model:      device.ModelT7,
				Ports:      device.Ports{Variant: device.T7Ports{}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(MatchError(fmt.Sprintf(
				"rack: rack %d does not exist: validation error",
				missing,
			)))
		})
	})

	Describe("Apply", func() {
		set := func(path string, value any) device.Action {
			return device.NewSetAction(device.SetPayload{
				Path:  []string{path},
				Value: value,
			})
		}
		configuration := func(ports device.PortsVariant) []device.Action {
			return []device.Action{
				set("name", "Renamed"),
				set("identifier", "dev1"),
				set("ports", device.Ports{Variant: ports}),
			}
		}
		identity := func(
			d device.Device,
			change func(*device.Device),
		) []device.Action {
			change(&d)
			return []device.Action{
				set("rack", d.Rack),
				set("serial", d.Serial),
				set("connection", d.Connection),
				set("model", d.Model),
			}
		}

		It("Should write the configuration and keep the identity", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				d.Key,
				configuration(device.T7Ports{Mux80Attached: true})...,
			)).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Name).To(Equal("Renamed"))
			Expect(got.Identifier).To(Equal("dev1"))
			Expect(got.Ports.Variant).To(HaveField("Mux80Attached", BeTrue()))
			Expect(got.Rack).To(Equal(d.Rack))
			Expect(got.Serial).To(Equal(d.Serial))
			Expect(got.Connection).To(Equal(d.Connection))
		})

		DescribeTable("Should write the port maps of each model",
			func(
				ctx SpecContext,
				model device.Model,
				bindPorts func(ch []channel.Key) device.PortsVariant,
			) {
				ports := bindPorts(createChannels(ctx, 2))
				d := newModel(model, nil)
				d.Ports = device.Ports{Variant: ports}
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
				Expect(svc.NewWriter(tx).Apply(ctx, d.Key, configuration(ports)...)).
					To(Succeed())
				Expect(retrieve(ctx, d.Key).Ports).
					To(Equal(device.Ports{Variant: ports}))
			},
			Entry("T4", device.ModelT4, func(ch []channel.Key) device.PortsVariant {
				return device.T4Ports{
					AnalogInput:  map[device.T4AnalogInputPort]channel.Key{},
					AnalogOutput: map[device.AnalogOutputPort]device.CommandStatePair{},
					DigitalInput: map[device.T4DigitalPort]channel.Key{},
					DigitalOutput: map[device.T4DigitalPort]device.CommandStatePair{
						device.T4DigitalPortDio4: {Command: ch[0], State: new(ch[1])},
					},
				}
			}),
			Entry("T7", device.ModelT7, func(ch []channel.Key) device.PortsVariant {
				return device.T7Ports{
					AnalogInput: map[device.T7AnalogInputPort]channel.Key{
						device.T7AnalogInputPortAin0: ch[0],
					},
					AnalogOutput:  map[device.AnalogOutputPort]device.CommandStatePair{},
					DigitalInput:  map[device.T7DigitalPort]channel.Key{},
					DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{},
				}
			}),
			Entry("T8", device.ModelT8, func(ch []channel.Key) device.PortsVariant {
				return device.T8Ports{
					AnalogInput: map[device.T8AnalogInputPort]channel.Key{
						device.T8AnalogInputPortAin7: ch[0],
					},
					AnalogOutput:  map[device.AnalogOutputPort]device.CommandStatePair{},
					DigitalInput:  map[device.T8DigitalPort]channel.Key{},
					DigitalOutput: map[device.T8DigitalPort]device.CommandStatePair{},
				}
			}),
		)

		DescribeTable(
			"Should refuse a pin with two roles",
			func(
				ctx SpecContext,
				ports device.PortsVariant,
				model device.Model,
				message string,
			) {
				d := newModel(model, ports)
				Expect(svc.NewWriter(tx).Create(ctx, &d)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("a T4 analog input on a mapped digital pin", device.T4Ports{
				AnalogInput: map[device.T4AnalogInputPort]channel.Key{
					device.T4AnalogInputPortAin4: 1,
				},
				DigitalInput: map[device.T4DigitalPort]channel.Key{
					device.T4DigitalPortDio4: 2,
				},
			}, device.ModelT4, "ports.analog_input: AIN4 shares its pin with DIO4"),
			Entry("a T7 digital port that is an input and an output", device.T7Ports{
				DigitalInput: map[device.T7DigitalPort]channel.Key{
					device.T7DigitalPortDio0: 1,
				},
				DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
					device.T7DigitalPortDio0: {Command: 2},
				},
			}, device.ModelT7, "ports.digital_output: DIO0 is also a digital input"),
			Entry("a T8 digital port that is an input and an output", device.T8Ports{
				DigitalInput: map[device.T8DigitalPort]channel.Key{
					device.T8DigitalPortDio0: 1,
				},
				DigitalOutput: map[device.T8DigitalPort]device.CommandStatePair{
					device.T8DigitalPortDio0: {Command: 2},
				},
			}, device.ModelT8, "ports.digital_output: DIO0 is also a digital input"),
		)

		It("Should accept a T4 analog input on a pin with no digital role", func(
			ctx SpecContext,
		) {
			ch := createChannels(ctx, 2)
			d := newModel(device.ModelT4, device.T4Ports{
				AnalogInput: map[device.T4AnalogInputPort]channel.Key{
					device.T4AnalogInputPortAin4: ch[0],
				},
				DigitalInput: map[device.T4DigitalPort]channel.Key{
					device.T4DigitalPortDio5: ch[1],
				},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
		})

		It("Should reject a port the model does not have in a configuration", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, configuration(
				device.T7Ports{
					DigitalInput: map[device.T7DigitalPort]channel.Key{"DIO23": 3},
				},
			)...)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("invalid digital_input key: DIO23")),
			))
		})

		It("Should refuse a Mux80 port in a configuration without a Mux80", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, configuration(
				device.T7Ports{
					AnalogInput: map[device.T7AnalogInputPort]channel.Key{
						device.T7AnalogInputPortAin127: 3,
					},
				},
			)...)).To(MatchError(ContainSubstring("AIN127 needs mux80_attached")))
		})

		It("Should write the identity and keep the configuration", func(
			ctx SpecContext,
		) {
			d := edit(create(ctx), func(b *device.Device) { b.Identifier = "dev1" })
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, identity(
				d,
				func(i *device.Device) {
					i.Rack = other.Key
					i.Serial = d.Serial + 1000
					i.Connection = device.ConnectionWifi
				},
			)...)).To(Succeed())
			got := retrieve(ctx, d.Key)
			Expect(got.Rack).To(Equal(other.Key))
			Expect(got.Serial).To(Equal(d.Serial + 1000))
			Expect(got.Connection).To(Equal(device.ConnectionWifi))
			Expect(got.Name).To(Equal(d.Name))
			Expect(got.Identifier).To(Equal("dev1"))
			Expect(retrieveStatus(ctx, d.Key).Details.Rack).To(Equal(other.Key))
		})

		It("Should refuse a move to a missing rack", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, identity(
				d,
				func(i *device.Device) { i.Rack = rackKey + 1000 },
			)...)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("does not exist")),
			))
			Expect(retrieve(ctx, d.Key).Rack).To(Equal(rackKey))
		})

		It("Should rename the status with the device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				d.Key,
				configuration(device.T7Ports{})...,
			)).To(Succeed())
			Expect(retrieveStatus(ctx, d.Key).Name).To(Equal("Renamed"))
		})

		It("Should write an unknown status when the status is missing", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(stat.NewWriter(tx).DeleteOwned(
				ctx,
				device.ResourceID(d.Key),
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				d.Key,
				configuration(device.T7Ports{})...,
			)).To(Succeed())
			s := retrieveStatus(ctx, d.Key)
			Expect(s.Name).To(Equal("Renamed"))
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Status unknown"))
			Expect(
				s.Details,
			).To(Equal(device.StatusDetails{Rack: rackKey, Device: d.Key}))
		})

		It("Should reject an invalid configuration", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, set("name", ""))).
				To(MatchError(ContainSubstring("name: required")))
		})

		DescribeTable(
			"Should refuse a different model",
			func(
				ctx SpecContext,
				acts func(device.Device) []device.Action,
				message string,
			) {
				d := create(ctx)
				Expect(svc.NewWriter(tx).Apply(ctx, d.Key, acts(d)...)).
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(message)),
					))
				Expect(retrieve(ctx, d.Key).Model).To(Equal(device.ModelT7))
			},
			Entry("in the ports", func(device.Device) []device.Action {
				return configuration(device.T4Ports{})
			}, "ports: are for t4, not t7"),
			Entry("in the model", func(d device.Device) []device.Action {
				return identity(
					d,
					func(i *device.Device) { i.Model = device.ModelT4 },
				)
			}, "model: cannot change from t7 to t4"),
		)

		It("Should reject a path that is not a device field", func(
			ctx SpecContext,
		) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Apply(ctx, d.Key, set("key", uuid.New()))).
				To(MatchError(validate.ErrValidation))
		})

		It("Should set a channel map of the ports", func(ctx SpecContext) {
			d := create(ctx)
			ch := createChannels(ctx, 1)
			inputs := map[device.T7DigitalPort]channel.Key{
				device.T7DigitalPortDio2: ch[0],
			}
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				d.Key,
				device.NewSetAction(device.SetPayload{
					Path:  []string{"ports", "digital_input"},
					Value: inputs,
				}),
			)).To(Succeed())
			Expect(retrieve(ctx, d.Key).Ports.Variant).
				To(HaveField("DigitalInput", Equal(inputs)))
		})

		It("Should return not found for a missing device", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(
				ctx,
				uuid.New(),
				configuration(device.T7Ports{})...,
			)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should remove the device and its status", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.NewWriter(tx).Delete(ctx, d.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(device.MatchKeys(d.Key)).
				Exists(ctx, tx)).To(BeFalse())
			var s device.Status
			Expect(stat.NewRetrieve[device.StatusDetails]().
				Where(status.MatchOwners[device.StatusDetails](device.ResourceID(d.Key))).
				Entry(&s).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should keep a rack that holds a device from deletion", func(
			ctx SpecContext,
		) {
			other := rack.Rack{Name: "Other"}
			Expect(rackSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			d := edit(create(ctx), func(b *device.Device) { b.Rack = other.Key })
			Expect(svc.NewWriter(tx).Create(ctx, &d)).To(Succeed())
			Expect(rackSvc.NewWriter(tx).Delete(ctx, nil, other.Key)).
				To(MatchError(ContainSubstring("labjack_device")))
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
	})

	Describe("ResourceID", func() {
		It("Should name a device by its key under the LabJack device type", func() {
			key := uuid.New()
			Expect(device.ResourceID(key)).To(Equal(resource.ID{
				Type: resource.TypeLabjackDevice,
				Key:  key.String(),
			}))
		})

		It("Should match the ID of a stored device", func(ctx SpecContext) {
			d := create(ctx)
			Expect(device.ResourceID(d.Key)).To(Equal(d.ResourceID()))
		})
	})

	Describe("Resources", func() {
		It("Should name each device by its name", func(ctx SpecContext) {
			d := create(ctx)
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{d.ResourceID()})).
				To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Name}}))
		})

		It("Should reject a key that is not a UUID", func(ctx SpecContext) {
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{{
				Type: resource.TypeLabjackDevice,
				Key:  "not-a-uuid",
			}})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should find a device by its name", func(ctx SpecContext) {
			d := create(ctx)
			res := MustSucceed(svc.Resources().Search(
				ctx,
				tx,
				search.Query[resource.ID]{Term: d.Name, Limit: 10},
			))
			Expect(res.Keys).To(ContainElement(d.ResourceID()))
		})
	})
})
