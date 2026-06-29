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
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Dependencies", Ordered, func() {
	var (
		node       mock.Node
		channelSvc *channel.Service
	)

	BeforeAll(func(ctx SpecContext) {
		node = mock.NewNode(ctx)
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       node.DB,
			Ontology: node.Ontology,
			Group:    node.Group,
			Search:   node.Search,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Ontology: node.Ontology,
			Group:    node.Group,
			Label:    labelSvc,
			Search:   node.Search,
		}))
		channelSvc = MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
			DB:           node.DB,
			Distribution: node.Channel,
			Status:       statusSvc,
		}))
	})

	Describe("NewDependencies", func() {
		It("Should build dependencies with read node channels", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     "sensor_1",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(node.Channel.Create(ctx, ch)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: types.Channels{
								Read: map[uint32]string{uint32(ch.Key()): "sensor_1"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads.Contains(ch.Key())).To(BeTrue())
			Expect(deps.Writes.Contains(ch.Key())).To(BeFalse())
			Expect(deps.ChannelDigests).To(HaveLen(1))
			Expect(deps.ChannelDigests[0].Key).To(Equal(uint32(ch.Key())))
			Expect(deps.ChannelDigests[0].DataType).To(Equal(telem.Float32T))
		})

		It("Should add channels from write nodes to writes set", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     "actuator_1",
				Virtual:  true,
				DataType: telem.Float64T,
			}
			Expect(node.Channel.Create(ctx, ch)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "write_node",
							Type: "write",
							Channels: types.Channels{
								Write: map[uint32]string{uint32(ch.Key()): "actuator_1"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Writes.Contains(ch.Key())).To(BeTrue())
			Expect(deps.Reads.Contains(ch.Key())).To(BeFalse())
		})

		It("Should add Channels.Write to writes set", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     "output_1",
				Virtual:  true,
				DataType: telem.Int32T,
			}
			Expect(node.Channel.Create(ctx, ch)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "any_node",
							Type: "constant",
							Channels: types.Channels{
								Write: map[uint32]string{uint32(ch.Key()): "output_1"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Writes.Contains(ch.Key())).To(BeTrue())
			Expect(deps.Reads.Contains(ch.Key())).To(BeFalse())
		})

		It("Should track index channels for reads", func(ctx SpecContext) {
			indexCh := &channel.Channel{
				Name:     "time_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
				Virtual:  false,
			}
			Expect(node.Channel.Create(ctx, indexCh)).To(Succeed())

			dataCh := &channel.Channel{
				Name:       "data_with_index",
				Virtual:    false,
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(node.Channel.Create(ctx, dataCh)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: types.Channels{
								Read: map[uint32]string{uint32(dataCh.Key()): "data_with_index"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads.Contains(dataCh.Key())).To(BeTrue())
			Expect(deps.Reads.Contains(indexCh.Key())).To(BeTrue())
			Expect(deps.ChannelDigests).To(HaveLen(2))
		})

		It("Should track index channels for writes", func(ctx SpecContext) {
			indexCh := &channel.Channel{
				Name:     "write_time_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
				Virtual:  false,
			}
			Expect(node.Channel.Create(ctx, indexCh)).To(Succeed())

			dataCh := &channel.Channel{
				Name:       "write_data_with_index",
				Virtual:    false,
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(node.Channel.Create(ctx, dataCh)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "write_node",
							Type: "write",
							Channels: types.Channels{
								Write: map[uint32]string{uint32(dataCh.Key()): "write_data_with_index"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Writes.Contains(dataCh.Key())).To(BeTrue())
			Expect(deps.Writes.Contains(indexCh.Key())).To(BeTrue())
		})

		It("Should handle nodes with both read and write channels", func(ctx SpecContext) {
			readCh := &channel.Channel{
				Name:     "input_sensor",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(node.Channel.Create(ctx, readCh)).To(Succeed())

			writeCh := &channel.Channel{
				Name:     "output_actuator",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(node.Channel.Create(ctx, writeCh)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "mixed_node",
							Type: "transform",
							Channels: types.Channels{
								Read:  map[uint32]string{uint32(readCh.Key()): "input_sensor"},
								Write: map[uint32]string{uint32(writeCh.Key()): "output_actuator"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads.Contains(readCh.Key())).To(BeTrue())
			Expect(deps.Writes.Contains(writeCh.Key())).To(BeTrue())
			Expect(deps.ChannelDigests).To(HaveLen(2))
		})

		It("Should handle multiple nodes with overlapping channels", func(ctx SpecContext) {
			sharedCh := &channel.Channel{
				Name:     "shared_channel",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(node.Channel.Create(ctx, sharedCh)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "node_1",
							Type: "on",
							Channels: types.Channels{
								Read: map[uint32]string{uint32(sharedCh.Key()): "shared_channel"},
							},
						},
						{
							Key:  "node_2",
							Type: "on",
							Channels: types.Channels{
								Read: map[uint32]string{uint32(sharedCh.Key()): "shared_channel"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads.Contains(sharedCh.Key())).To(BeTrue())
			Expect(deps.ChannelDigests).To(HaveLen(1))
		})

		It("Should handle empty module", func(ctx SpecContext) {
			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{},
				},
			}
			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads).To(BeEmpty())
			Expect(deps.Writes).To(BeEmpty())
			Expect(deps.ChannelDigests).To(BeEmpty())
		})

		It("Should handle module with nodes that have no channels", func(ctx SpecContext) {
			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:      "constant_node",
							Type:     "constant",
							Channels: types.Channels{},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads).To(BeEmpty())
			Expect(deps.Writes).To(BeEmpty())
			Expect(deps.ChannelDigests).To(BeEmpty())
		})

		It("Should return error when channel retrieval fails", func(ctx SpecContext) {
			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "invalid_node",
							Type: "on",
							Channels: types.Channels{
								Read: map[uint32]string{999999: "nonexistent"},
							},
						},
					},
				},
			}

			Expect(runtime.NewDependencies(ctx, channelSvc, prog)).
				Error().To(MatchError(query.ErrNotFound))
		})

		It("Should not add index channel to sets when channel is virtual", func(ctx SpecContext) {
			virtualCh := &channel.Channel{
				Name:       "virtual_no_index",
				Virtual:    true,
				DataType:   telem.Float32T,
				LocalIndex: 0,
			}
			Expect(node.Channel.Create(ctx, virtualCh)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: types.Channels{
								Read: map[uint32]string{uint32(virtualCh.Key()): "virtual_no_index"},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads.Contains(virtualCh.Key())).To(BeTrue())
			Expect(deps.ChannelDigests).To(HaveLen(1))
		})

		It("Should handle interval-triggered function with stateful variable writing to channel", func(ctx SpecContext) {
			virtCh := &channel.Channel{
				Name:     "virt_stateful_test",
				Virtual:  true,
				DataType: telem.Float32T,
			}
			Expect(node.Channel.Create(ctx, virtCh)).To(Succeed())

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

			resolver := channelSvc.NewArcSymbolResolver(nil)
			compiled := MustSucceed(arc.CompileText(ctx, prog, arc.NewRoot(resolver)))

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, compiled))
			Expect(deps.Reads).To(BeEmpty())
			Expect(deps.Writes.Contains(virtCh.Key())).To(BeTrue())
			Expect(deps.Writes).To(HaveLen(1))
			Expect(deps.ChannelDigests).To(HaveLen(1))
			Expect(deps.ChannelDigests[0].Key).To(Equal(uint32(virtCh.Key())))
			Expect(deps.ChannelDigests[0].DataType).To(Equal(telem.Float32T))
		})

		It("Should add dynamic set_authority channel to writes even if never written to", func(ctx SpecContext) {
			triggerCh := &channel.Channel{
				Name:     "dyn_auth_trigger",
				Virtual:  true,
				DataType: telem.Uint8T,
			}
			Expect(node.Channel.Create(ctx, triggerCh)).To(Succeed())

			valveCh := &channel.Channel{
				Name:     "dyn_auth_valve",
				Virtual:  true,
				DataType: telem.Uint8T,
			}
			Expect(node.Channel.Create(ctx, valveCh)).To(Succeed())

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

			resolver := channelSvc.NewArcSymbolResolver(nil)
			compiled := MustSucceed(arc.CompileText(ctx, prog, arc.NewRoot(resolver)))

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, compiled))
			Expect(deps.Writes.Contains(valveCh.Key())).To(BeTrue(),
				"channel referenced only in set_authority config should be in writes")
		})

		It("Should add authority-declared channels to writes even if not in any node", func(ctx SpecContext) {
			authOnlyCh := &channel.Channel{
				Name:     "authority_only_ch",
				Virtual:  true,
				DataType: telem.Float64T,
			}
			Expect(node.Channel.Create(ctx, authOnlyCh)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Authorities: ir.Authorities{
						Channels: map[uint32]uint8{
							uint32(authOnlyCh.Key()): 100,
						},
					},
					Nodes: []ir.Node{},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Writes.Contains(authOnlyCh.Key())).To(BeTrue())
			Expect(deps.ChannelDigests).To(HaveLen(1))
			Expect(deps.ChannelDigests[0].Key).To(Equal(uint32(authOnlyCh.Key())))
		})

		It("Should build complete dependencies with complex module", func(ctx SpecContext) {
			indexCh := &channel.Channel{
				Name:     "complex_index",
				DataType: telem.TimeStampT,
				IsIndex:  true,
				Virtual:  false,
			}
			Expect(node.Channel.Create(ctx, indexCh)).To(Succeed())

			readCh1 := &channel.Channel{
				Name:       "complex_read_1",
				Virtual:    false,
				DataType:   telem.Float32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(node.Channel.Create(ctx, readCh1)).To(Succeed())

			readCh2 := &channel.Channel{
				Name:     "complex_read_2",
				Virtual:  true,
				DataType: telem.Float64T,
			}
			Expect(node.Channel.Create(ctx, readCh2)).To(Succeed())

			writeCh := &channel.Channel{
				Name:       "complex_write",
				Virtual:    false,
				DataType:   telem.Int32T,
				LocalIndex: indexCh.LocalKey,
			}
			Expect(node.Channel.Create(ctx, writeCh)).To(Succeed())

			prog := arc.Program{
				IR: ir.IR{
					Nodes: []ir.Node{
						{
							Key:  "read_node",
							Type: "on",
							Channels: types.Channels{
								Read: map[uint32]string{
									uint32(readCh1.Key()): "complex_read_1",
									uint32(readCh2.Key()): "complex_read_2",
								},
							},
						},
						{
							Key:  "write_node",
							Type: "write",
							Channels: types.Channels{
								Write: map[uint32]string{
									uint32(readCh1.Key()): "complex_read_1",
									uint32(writeCh.Key()): "complex_write",
								},
							},
						},
					},
				},
			}

			deps := MustSucceed(runtime.NewDependencies(ctx, channelSvc, prog))
			Expect(deps.Reads.Contains(readCh1.Key())).To(BeTrue())
			Expect(deps.Reads.Contains(readCh2.Key())).To(BeTrue())
			Expect(deps.Reads.Contains(indexCh.Key())).To(BeTrue())
			Expect(deps.Writes.Contains(readCh1.Key())).To(BeTrue())
			Expect(deps.Writes.Contains(writeCh.Key())).To(BeTrue())
			Expect(deps.Writes.Contains(indexCh.Key())).To(BeTrue())
			Expect(deps.ChannelDigests).To(HaveLen(4))
		})
	})
})
