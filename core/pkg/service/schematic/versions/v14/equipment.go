// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14

import (
	"encoding/json/v2"
	"reflect"

	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

const (
	activeState              = "active"
	normallyOpenSolenoidName = "normally_open_solenoid_valve"
)

// SymbolNode returns the node that draws the built-in symbol of a stored equipment
// node, with the same label, scale, color, timing, and channels. It returns false for a
// node with no built-in symbol.
func SymbolNode(old v13.NodeConfig) (SymbolNodeConfig, bool, error) {
	if v, ok := old.Variant.(v13.SolenoidValveNodeConfig); ok {
		n := toggle(v.ToggleSymbolConfig)
		n.Spec = builtin(string(v13.SolenoidValveNodeConfigType))
		if v.NormallyOpen {
			n.Spec = builtin(normallyOpenSolenoidName)
		}
		return n, true, nil
	}
	// Each other equipment variant holds its shape as its only field.
	v := reflect.ValueOf(old.Variant)
	if v.Kind() != reflect.Struct || v.NumField() != 1 {
		return SymbolNodeConfig{}, false, nil
	}
	var n SymbolNodeConfig
	switch s := v.Field(0).Interface().(type) {
	case v13.StaticSymbolConfig:
		n = base(s.LabeledConfig, s.ScaledConfig, s.StrokeColor)
	case v13.ToggleSymbolConfig:
		n = toggle(s)
	case v13.DummyToggleSymbolConfig:
		n = base(s.LabeledConfig, s.ScaledConfig, s.StrokeColor)
		value := 0.0
		if s.Enabled {
			value = 1
		}
		n.States = []StateBinding{{
			Key:    activeState,
			Source: &Source{Variant: LocalSource{Value: value}},
		}}
		n.Disabled = !s.Clickable
	default:
		return SymbolNodeConfig{}, false, nil
	}
	name, err := variantName(old)
	n.Spec = builtin(name)
	return n, true, err
}

// variantName returns the variant tag that the codec writes for cfg, which is the name
// of its built-in symbol.
func variantName(cfg v13.NodeConfig) (string, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	var tagged struct {
		Variant string `json:"variant"`
	}
	return tagged.Variant, json.Unmarshal(b, &tagged)
}

func builtin(name string) symbol.Ref {
	return symbol.Ref{Variant: symbol.BuiltinRef{Name: name}}
}

func base(
	label LabeledConfig,
	scale ScaledConfig,
	stroke *xcolor.Color,
) SymbolNodeConfig {
	var n SymbolNodeConfig
	n.SetDefaults()
	n.LabeledConfig, n.ScaledConfig = label, scale
	n.Color = fixed(stroke)
	n.States, n.Commands = []StateBinding{}, []CommandBinding{}
	return n
}

func toggle(c v13.ToggleSymbolConfig) SymbolNodeConfig {
	n := base(c.LabeledConfig, c.ScaledConfig, c.StrokeColor)
	n.StalenessTimeout = c.StalenessTimeout
	n.OnClickDelay = c.OnClickDelay
	n.Control = &c.Control
	n.States = []StateBinding{{
		Key: activeState,
		Source: &Source{Variant: ChannelSource{
			Key:     channelKey(c.StateChannel),
			Command: channelKey(c.CommandChannel),
		}},
	}}
	return n
}

// channelKey returns nil for 0, the key of no channel.
func channelKey(k channel.Key) *channel.Key {
	if k == 0 {
		return nil
	}
	return &k
}
