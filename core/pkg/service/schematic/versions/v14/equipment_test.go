// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	"github.com/synnaxlabs/x/defaults"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("SymbolNode", func() {
	convert := func(v v13.NodeConfigVariant) v14.SymbolNodeConfig {
		GinkgoHelper()
		n, ok := MustSucceed2(v14.SymbolNode(v13.NodeConfig{Variant: v}))
		Expect(ok).To(BeTrue())
		return n
	}

	// expected returns a node of the named built-in with no bindings and the defaults
	// of a stored node.
	expected := func(name string) v14.SymbolNodeConfig {
		n := defaults.Fill(v14.SymbolNodeConfig{})
		n.Spec = symbol.Ref{Variant: symbol.BuiltinRef{Name: name}}
		n.States, n.Commands = []v14.StateBinding{}, []v14.CommandBinding{}
		return n
	}

	active := func(source v14.SourceVariant) []v14.StateBinding {
		return []v14.StateBinding{{Key: "active", Source: &v14.Source{Variant: source}}}
	}

	It("Should draw a static picture as its built-in with no bindings", func() {
		in := defaults.Fill(v13.FilterNodeConfig{})
		in.Label.Label, in.Scale, in.StrokeColor = "Inlet filter", 1.5, shade(1)
		want := expected("filter")
		want.Label.Label, want.Scale, want.Color = "Inlet filter", 1.5, fixed(1)
		Expect(convert(in)).To(Equal(want))
	})

	It("Should bind the active state of a toggle to its channels", func() {
		in := defaults.Fill(v13.PumpNodeConfig{})
		in.StateChannel, in.CommandChannel = 1, 2
		in.StalenessTimeout, in.StalenessColor = 10*telem.Second, shade(3)
		in.OnClickDelay = 500 * telem.Millisecond
		in.Control.Authority = new(uint8(200))
		in.StrokeColor = shade(2)
		want := expected("pump")
		want.StalenessTimeout = 10 * telem.Second
		want.OnClickDelay = 500 * telem.Millisecond
		want.Control = &in.Control
		want.Color = fixed(2)
		state, command := channel.Key(1), channel.Key(2)
		want.States = active(v14.ChannelSource{Key: &state, Command: &command})
		Expect(convert(in)).To(Equal(want))
	})

	It("Should give a toggle with no channels a source with no keys", func() {
		in := defaults.Fill(v13.ValveNodeConfig{})
		Expect(convert(in).States).To(Equal(active(v14.ChannelSource{})))
	})

	DescribeTable("Should draw a solenoid valve as the built-in of its kind", func(
		normallyOpen bool,
		name string,
	) {
		in := defaults.Fill(v13.SolenoidValveNodeConfig{NormallyOpen: normallyOpen})
		in.StateChannel = 1
		n := convert(in)
		Expect(n.Spec.Variant).To(Equal(symbol.BuiltinRef{Name: name}))
		state := channel.Key(1)
		Expect(n.States).To(Equal(active(v14.ChannelSource{Key: &state})))
	},
		Entry("normally closed", false, "solenoid_valve"),
		Entry("normally open", true, "normally_open_solenoid_valve"),
	)

	DescribeTable("Should hold the state of a local valve in the node", func(
		enabled, clickable bool,
		value float64,
	) {
		in := defaults.Fill(v13.ManualValveNodeConfig{})
		in.Enabled, in.Clickable = enabled, clickable
		want := expected("manual_valve")
		want.States = active(v14.LocalSource{Value: value})
		want.Disabled = !clickable
		Expect(convert(in)).To(Equal(want))
	},
		Entry("off and fixed", false, false, 0.0),
		Entry("off and clickable", false, true, 0.0),
		Entry("on and fixed", true, false, 1.0),
		Entry("on and clickable", true, true, 1.0),
	)

	DescribeTable("Should leave a node with no built-in symbol", func(
		v v13.NodeConfigVariant,
	) {
		_, ok := MustSucceed2(v14.SymbolNode(v13.NodeConfig{Variant: v}))
		Expect(ok).To(BeFalse())
	},
		Entry("a light", v13.LightNodeConfig{Channel: 1}),
		Entry("a custom actuator", v13.CustomActuatorNodeConfig{SpecKey: "k"}),
		Entry("a custom static", v13.CustomStaticNodeConfig{SpecKey: "k"}),
		Entry("a box", v13.BoxNodeConfig{}),
		Entry("a group box with one field", v13.GroupBoxNodeConfig{Locked: true}),
	)
})
