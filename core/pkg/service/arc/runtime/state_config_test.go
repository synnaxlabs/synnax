// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	arcsymbol "github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("StateConfig", Ordered, func() {
	var dist mock.Node

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
	})

	AfterAll(func() {
		Expect(dist.Close()).To(Succeed())
	})

	Describe("NewStateConfig", func() {
		It("Should build config with read node channels", func() {
			ch := &channel.Channel{
				Name:     "sensor_1",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: arcsymbol.Channels{
								Read: set.Mapped[uint32, string]{uint32(ch.Key()): "sensor_1"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads.Contains(ch.Key())).To(BeTrue())
			Expect(cfg.Writes.Contains(ch.Key())).To(BeFalse())
			Expect(cfg.State.ChannelDigests).To(HaveLen(1))
			Expect(cfg.State.ChannelDigests[0].Key).To(Equal(uint32(ch.Key())))
			Expect(cfg.State.ChannelDigests[0].DataType).To(Equal(telem.Float32T))
		})

		It("Should add channels from write nodes to writes set", func() {
			ch := &channel.Channel{
				Name:     "actuator_1",
				Virtual:  true,
				DataType: telem.Float64T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "write_node",
							Type: "write",
							Channels: arcsymbol.Channels{
								Write: set.Mapped[uint32, string]{uint32(ch.Key()): "actuator_1"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Writes.Contains(ch.Key())).To(BeTrue())
			Expect(cfg.Reads.Contains(ch.Key())).To(BeFalse())
		})

		It("Should add Channels.Write to writes set", func() {
			ch := &channel.Channel{
				Name:     "output_1",
				Virtual:  true,
				DataType: telem.Int32T,
			}
			Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "any_node",
							Type: "constant",
							Channels: arcsymbol.Channels{
								Write: set.Mapped[uint32, string]{uint32(ch.Key()): "output_1"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Writes.Contains(ch.Key())).To(BeTrue())
			Expect(cfg.Reads.Contains(ch.Key())).To(BeFalse())
		})

		It("Should track index channels for reads", func() {
			indexCh := &channel.Channel{
				Name:     "time_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
				Virtual:  false,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())

			dataCh := &channel.Channel{
				Name:       "data_with_index",
				Virtual:    false,
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: arcsymbol.Channels{
								Read: set.Mapped[uint32, string]{uint32(dataCh.Key()): "data_with_index"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads.Contains(dataCh.Key())).To(BeTrue())
			Expect(cfg.Reads.Contains(indexCh.Key())).To(BeTrue())
			Expect(cfg.State.ChannelDigests).To(HaveLen(2))
		})

		It("Should track index channels for writes", func() {
			indexCh := &channel.Channel{
				Name:     "write_time_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
				Virtual:  false,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())

			dataCh := &channel.Channel{
				Name:       "write_data_with_index",
				Virtual:    false,
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, dataCh)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "write_node",
							Type: "write",
							Channels: arcsymbol.Channels{
								Write: set.Mapped[uint32, string]{uint32(dataCh.Key()): "write_data_with_index"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Writes.Contains(dataCh.Key())).To(BeTrue())
			Expect(cfg.Writes.Contains(indexCh.Key())).To(BeTrue())
		})

		It("Should handle nodes with both read and write channels", func() {
			readCh := &channel.Channel{
				Name:     "input_sensor",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, readCh)).To(Succeed())

			writeCh := &channel.Channel{
				Name:     "output_actuator",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, writeCh)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "mixed_node",
							Type: "transform",
							Channels: arcsymbol.Channels{
								Read:  set.Mapped[uint32, string]{uint32(readCh.Key()): "input_sensor"},
								Write: set.Mapped[uint32, string]{uint32(writeCh.Key()): "output_actuator"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads.Contains(readCh.Key())).To(BeTrue())
			Expect(cfg.Writes.Contains(writeCh.Key())).To(BeTrue())
			Expect(cfg.State.ChannelDigests).To(HaveLen(2))
		})

		It("Should handle multiple nodes with overlapping channels", func() {
			sharedCh := &channel.Channel{
				Name:     "shared_channel",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, sharedCh)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "node_1",
							Type: "on",
							Channels: arcsymbol.Channels{
								Read: set.Mapped[uint32, string]{uint32(sharedCh.Key()): "shared_channel"},
							},
						},
						{
							Key:  "node_2",
							Type: "on",
							Channels: arcsymbol.Channels{
								Read: set.Mapped[uint32, string]{uint32(sharedCh.Key()): "shared_channel"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads.Contains(sharedCh.Key())).To(BeTrue())
			Expect(cfg.State.ChannelDigests).To(HaveLen(1))
		})

		It("Should handle empty module", func() {
			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads).To(HaveLen(0))
			Expect(cfg.Writes).To(HaveLen(0))
			Expect(cfg.State.ChannelDigests).To(HaveLen(0))
		})

		It("Should handle module with nodes that have no channels", func() {
			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:      "constant_node",
							Type:     "constant",
							Channels: arcsymbol.Channels{},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads).To(HaveLen(0))
			Expect(cfg.Writes).To(HaveLen(0))
			Expect(cfg.State.ChannelDigests).To(HaveLen(0))
		})

		It("Should return error when channel retrieval fails", func() {
			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "invalid_node",
							Type: "on",
							Channels: arcsymbol.Channels{
								Read: set.Mapped[uint32, string]{999999: "nonexistent"},
							},
						},
					},
				},
			}

			_, err := runtime.NewStateConfig(ctx, dist.Channel, module)
			Expect(err).To(HaveOccurred())
		})

		It("Should not add index channel to sets when channel is virtual", func() {
			virtualCh := &channel.Channel{
				Name:       "virtual_no_index",
				Virtual:    true,
				DataType:   telem.Float32T,
				LocalIndex: 0,
			}
			Expect(dist.Channel.Create(ctx, virtualCh)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: arcsymbol.Channels{
								Read: set.Mapped[uint32, string]{uint32(virtualCh.Key()): "virtual_no_index"},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads.Contains(virtualCh.Key())).To(BeTrue())
			Expect(cfg.State.ChannelDigests).To(HaveLen(1))
		})

		It("Should handle interval-triggered function with stateful variable writing to channel", func() {
			virtCh := &channel.Channel{
				Name:     "virt_stateful_test",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(dist.Channel.Create(ctx, virtCh)).To(Succeed())

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					func cat() {
						counter f32 $= 1.0
						counter += 1.2
						%s = counter
					}
					interval{period=500ms} -> cat{}
				`, virtCh.Name),
			}

			resolver := symbol.CreateResolver(dist.Channel)
			module := MustSucceed(arc.CompileText(ctx, prog, arc.WithResolver(resolver)))

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads).To(HaveLen(0))
			Expect(cfg.Writes.Contains(virtCh.Key())).To(BeTrue())
			Expect(cfg.Writes).To(HaveLen(1))
			Expect(cfg.State.ChannelDigests).To(HaveLen(1))
			Expect(cfg.State.ChannelDigests[0].Key).To(Equal(uint32(virtCh.Key())))
			Expect(cfg.State.ChannelDigests[0].DataType).To(Equal(telem.Float32T))
		})

		It("Should add dynamic set_authority channel to writes even if never written to", func() {
			triggerCh := &channel.Channel{
				Name:     "dyn_auth_trigger",
				Virtual:  true,
				DataType: telem.Uint8T,
			}
			Expect(dist.Channel.Create(ctx, triggerCh)).To(Succeed())

			valveCh := &channel.Channel{
				Name:     "dyn_auth_valve",
				Virtual:  true,
				DataType: telem.Uint8T,
			}
			Expect(dist.Channel.Create(ctx, valveCh)).To(Succeed())

			prog := arc.Text{
				Raw: fmt.Sprintf(`
					sequence seq {
						stage claim {
							1 -> set_authority{value=100, channel=%s}
						}
					}
					%s => seq
				`, valveCh.Name, triggerCh.Name),
			}

			resolver := symbol.CreateResolver(dist.Channel)
			module := MustSucceed(arc.CompileText(ctx, prog, arc.WithResolver(resolver)))

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Writes.Contains(valveCh.Key())).To(BeTrue(),
				"channel referenced only in set_authority config should be in writes")
		})

		It("Should add authority-declared channels to writes even if not in any node", func() {
			authOnlyCh := &channel.Channel{
				Name:     "authority_only_ch",
				Virtual:  true,
				DataType: telem.Float64T,
			}
			Expect(dist.Channel.Create(ctx, authOnlyCh)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Authorities: ir.Authorities{
						Channels: map[uint32]uint8{
							uint32(authOnlyCh.Key()): 100,
						},
					},
					Nodes: []ir.Node{},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Writes.Contains(authOnlyCh.Key())).To(BeTrue())
			Expect(cfg.State.ChannelDigests).To(HaveLen(1))
			Expect(cfg.State.ChannelDigests[0].Key).To(Equal(uint32(authOnlyCh.Key())))
		})

		It("Should build complete config with complex module", func() {
			indexCh := &channel.Channel{
				Name:     "complex_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
				Virtual:  false,
			}
			Expect(dist.Channel.Create(ctx, indexCh)).To(Succeed())

			readCh1 := &channel.Channel{
				Name:       "complex_read_1",
				Virtual:    false,
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, readCh1)).To(Succeed())

			readCh2 := &channel.Channel{
				Name:     "complex_read_2",
				Virtual:  true,
				DataType: telem.Float64T,
			}
			Expect(dist.Channel.Create(ctx, readCh2)).To(Succeed())

			writeCh := &channel.Channel{
				Name:       "complex_write",
				Virtual:    false,
				DataType:   telem.Int32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, writeCh)).To(Succeed())

			module := arc.Module{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: arcsymbol.Channels{
								Read: set.Mapped[uint32, string]{
									uint32(readCh1.Key()): "complex_read_1",
									uint32(readCh2.Key()): "complex_read_2",
								},
							},
						},
						{
							Key:  "write_node",
							Type: "write",
							Channels: arcsymbol.Channels{
								Write: set.Mapped[uint32, string]{
									uint32(readCh1.Key()): "complex_read_1",
									uint32(writeCh.Key()): "complex_write",
								},
							},
						},
					},
				},
			}

			cfg := MustSucceed(runtime.NewStateConfig(ctx, dist.Channel, module))
			Expect(cfg.Reads.Contains(readCh1.Key())).To(BeTrue())
			Expect(cfg.Reads.Contains(readCh2.Key())).To(BeTrue())
			Expect(cfg.Reads.Contains(indexCh.Key())).To(BeTrue())
			Expect(cfg.Writes.Contains(readCh1.Key())).To(BeTrue())
			Expect(cfg.Writes.Contains(writeCh.Key())).To(BeTrue())
			Expect(cfg.Writes.Contains(indexCh.Key())).To(BeTrue())
			Expect(cfg.State.ChannelDigests).To(HaveLen(4))
		})
	})
})
