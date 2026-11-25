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
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Uses", func() {
	var (
		u   user.User
		ch  channel.Channel
		tsk task.Task
		sch schematic.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		ch = createChannel(ctx, telem.Float32T)
		dev := device.Device{
			Key:      UniqueChannelName(),
			Rack:     layer.Rack.EmbeddedKey,
			Name:     "plc",
			Make:     "modbus",
			Model:    "modbus",
			Location: "127.0.0.1:502",
		}
		proj := project.Project{Name: "Uses"}
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = ch.Key()
		sch = schematic.Schematic{
			Name:    "pid",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		tsk = task.Task{
			Name: "modbus_read",
			Type: "modbus_read",
			Config: msgpack.EncodedJSON{
				"device": dev.Key,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": ch.Key(),
				}},
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
				return err
			}
			if err := layer.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
				return err
			}
			if err := layer.Schematic.NewWriter(tx).Create(ctx, &sch); err != nil {
				return err
			}
			return layer.Task.NewWriter(tx).Create(ctx, &tsk)
		})).To(Succeed())
	})

	grant := func(ctx SpecContext, types ...resource.Type) {
		GinkgoHelper()
		ids := make([]resource.ID, len(types))
		for i, t := range types {
			ids[i] = resource.ID{Type: t}
		}
		grantOn(ctx, u.ResourceID(), []access.Action{access.ActionRetrieve}, ids...)
	}

	retrieve := func(ctx SpecContext, uses bool) apichannel.Channel {
		GinkgoHelper()
		res := MustSucceed(retrieveChannels(
			AuthedCtx(ctx, u),
			apichannel.RetrieveRequest{
				Where: &channel.Where{
					Key: &filter.Set[channel.Key]{Eq: new(ch.Key())},
				},
				IncludeUses: uses,
			},
		))
		Expect(res.Channels).To(HaveLen(1))
		return res.Channels[0]
	}

	It(
		"Should list each record that uses the channel, with a task config as its task",
		func(
			ctx SpecContext,
		) {
			grant(ctx, resource.TypeChannel, resource.TypeTask, resource.TypeSchematic)
			Expect(retrieve(ctx, true).Uses).To(ConsistOf(
				apichannel.Use{ID: tsk.ResourceID(), Direction: channel.DirectionWrite},
				apichannel.Use{ID: sch.ResourceID(), Direction: channel.DirectionRead},
			))
		},
	)

	It("Should list an Arc and its deployed task, through its drift record, as uses",
		func(ctx SpecContext) {
			grant(ctx, resource.TypeChannel, resource.TypeTask, resource.TypeArc)
			out := createChannel(ctx, telem.Float32T)
			a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
				Name: uuid.New().String(),
				Source: arc.Source{Variant: arc.TextSource{
					Text: text.Text{Raw: ch.Name + " -> " + out.Name},
				}},
			}}}
			var deployed *task.Task
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				w := layer.Arc.NewWriter(tx)
				if err = w.Create(ctx, &a); err != nil {
					return err
				}
				deployed, err = w.UpdateTask(
					ctx,
					a.Base().Key,
					arc.TaskUpdate{Rack: new(layer.Rack.EmbeddedKey)},
				)
				return err
			})).To(Succeed())
			// The Driver takes the program at its configure.
			MustSucceed(layer.Arc.DeployProgram(
				ctx,
				a.Base().Key,
				layer.Rack.EmbeddedKey,
			))
			Expect(retrieve(ctx, true).Uses).To(ContainElements(
				apichannel.Use{ID: a.ResourceID(), Direction: channel.DirectionRead},
				apichannel.Use{
					ID:        deployed.ResourceID(),
					Direction: channel.DirectionRead,
				},
			))
		})

	It("Should leave out a record the caller cannot retrieve", func(ctx SpecContext) {
		grant(ctx, resource.TypeChannel, resource.TypeTask)
		Expect(retrieve(ctx, true).Uses).To(ConsistOf(
			apichannel.Use{ID: tsk.ResourceID(), Direction: channel.DirectionWrite},
		))
	})

	It("Should return empty uses for a channel that nothing uses", func(
		ctx SpecContext,
	) {
		grant(ctx, resource.TypeChannel)
		unused := createChannel(ctx, telem.Float32T)
		res := MustSucceed(retrieveChannels(
			AuthedCtx(ctx, u),
			apichannel.RetrieveRequest{
				Where: &channel.Where{
					Key: &filter.Set[channel.Key]{Eq: new(unused.Key())},
				},
				IncludeUses: true,
			},
		))
		Expect(res.Channels).To(HaveLen(1))
		Expect(res.Channels[0].Uses).ToNot(BeNil())
		Expect(res.Channels[0].Uses).To(BeEmpty())
	})

	It("Should leave uses out when the request does not ask for them", func(
		ctx SpecContext,
	) {
		grant(ctx, resource.TypeChannel, resource.TypeTask, resource.TypeSchematic)
		Expect(retrieve(ctx, false).Uses).To(BeNil())
	})
})

var _ = Describe("Built-in channels", func() {
	var builtIn channel.Channel
	BeforeEach(func(ctx SpecContext) {
		builtIn = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.JSONT,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
	})

	It("Should report a built-in channel as internal", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			channel.ResourceID(builtIn.Key()),
		)
		res := MustSucceed(
			retrieveChannels(AuthedCtx(ctx, u), apichannel.RetrieveRequest{
				Where: &channel.Where{
					Key: &filter.Set[channel.Key]{Eq: new(builtIn.Key())},
				},
			}),
		)
		Expect(res.Channels).To(HaveLen(1))
		Expect(res.Channels[0].Internal).To(BeTrue())
	})

	retrieveInternal := func(ctx SpecContext, u user.User) []apichannel.Channel {
		GinkgoHelper()
		return MustSucceed(retrieveChannels(
			AuthedCtx(ctx, u),
			apichannel.RetrieveRequest{
				Where: &channel.Where{Internal: &filter.Bool{Eq: new(true)}},
			},
		)).Channels
	}

	It("Should list each built-in channel through the internal filter", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
		)
		plain := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.JSONT,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &plain)
		})).To(Succeed())
		chs := retrieveInternal(ctx, u)
		keys := make([]channel.Key, 0, len(chs))
		for _, ch := range chs {
			Expect(ch.Internal).To(BeTrue())
			keys = append(keys, ch.Key)
		}
		Expect(keys).To(ContainElement(builtIn.Key()))
		Expect(keys).ToNot(ContainElement(plain.Key()))
	})

	It("Should leave out the built-in channels a subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		Expect(retrieveInternal(ctx, u)).To(BeEmpty())
	})
})
