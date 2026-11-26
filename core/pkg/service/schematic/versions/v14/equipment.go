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
	"context"
	"encoding/json/v2"
	"reflect"
	"slices"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	enumeration "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	"go.uber.org/zap"
)

const (
	activeState              = "active"
	normallyOpenSolenoidName = "normally_open_solenoid_valve"
)

// builtinWithin is the match of the active state of each built-in toggle and light.
var builtinWithin = spatial.Bounds{Lower: 0.9, Upper: 1.1}

// SymbolNode returns the symbol node that a stored equipment node becomes, with the
// same label, scale, color, timing, and channels. A custom node draws the custom symbol
// of its spec key. It returns false for a node with no symbol, and for a light with an
// on color or a custom node with state overrides, since those need new records.
func SymbolNode(old v13.NodeConfig) (SymbolNodeConfig, bool, error) {
	switch v := old.Variant.(type) {
	case v13.SolenoidValveNodeConfig:
		n := toggle(v.ToggleConfig, v.StrokeColor)
		n.Spec = builtin(string(v13.SolenoidValveNodeConfigType))
		if v.NormallyOpen {
			n.Spec = builtin(normallyOpenSolenoidName)
		}
		return n, true, nil
	case v13.LightNodeConfig:
		if v.OnColor != nil {
			return SymbolNodeConfig{}, false, nil
		}
		n := base(v.LabeledConfig, v.ScaledConfig, v.StrokeColor)
		n.Spec = builtin(string(v13.LightNodeConfigType))
		n.StalenessTimeout = v.StalenessTimeout
		n.States = []StateBinding{{
			Key:    activeState,
			Source: &Source{Variant: ChannelSource{Key: channelKey(v.Channel)}},
			Match:  lightMatch(v.Threshold),
		}}
		return n, true, nil
	case v13.CustomActuatorNodeConfig:
		if len(v.StateOverrides) > 0 {
			return SymbolNodeConfig{}, false, nil
		}
		n := toggle(v.ToggleConfig, nil)
		n.Spec = custom(v.SpecKey)
		return n, true, nil
	case v13.CustomStaticNodeConfig:
		if len(v.StateOverrides) > 0 {
			return SymbolNodeConfig{}, false, nil
		}
		n := base(v.LabeledConfig, v.ScaledConfig, nil)
		n.Spec = custom(v.SpecKey)
		n.States = []StateBinding{{Key: activeState}}
		return n, true, nil
	}
	return shaped(old)
}

// shaped converts an equipment variant that holds its shape as its only field.
func shaped(old v13.NodeConfig) (SymbolNodeConfig, bool, error) {
	v := reflect.ValueOf(old.Variant)
	if v.Kind() != reflect.Struct || v.NumField() != 1 {
		return SymbolNodeConfig{}, false, nil
	}
	var n SymbolNodeConfig
	switch s := v.Field(0).Interface().(type) {
	case v13.StaticSymbolConfig:
		n = base(s.LabeledConfig, s.ScaledConfig, s.StrokeColor)
	case v13.ToggleSymbolConfig:
		n = toggle(s.ToggleConfig, s.StrokeColor)
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
	if err != nil {
		return SymbolNodeConfig{}, false, err
	}
	n.Spec = builtin(name)
	return n, true, nil
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
	if err = json.Unmarshal(b, &tagged); err != nil {
		return "", err
	}
	return tagged.Variant, nil
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

// custom returns the ref of the custom symbol with key, or a missing ref when key is
// not a symbol key.
func custom(key string) symbol.Ref {
	k, err := uuid.Parse(key)
	if err != nil {
		return symbol.Ref{Variant: symbol.MissingRef{Key: key}}
	}
	return symbol.Ref{Variant: symbol.CustomRef{Key: k}}
}

// lightMatch returns the match of a light's threshold, swapped when inverted, or nil
// when the light has none, when it is empty, or when it equals the built-in's match.
func lightMatch(threshold *spatial.Bounds) *symbol.Matcher {
	if threshold == nil {
		return nil
	}
	b := *threshold
	if b.Lower > b.Upper {
		b.Lower, b.Upper = b.Upper, b.Lower
	}
	if b.Lower == b.Upper || b == builtinWithin {
		return nil
	}
	return &symbol.Matcher{Variant: symbol.WithinMatcher{Bounds: b}}
}

func toggle(c v13.ToggleConfig, stroke *xcolor.Color) SymbolNodeConfig {
	n := base(c.LabeledConfig, c.ScaledConfig, stroke)
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

// actuate makes each stored symbol that a custom_actuator node of s draws an actuator,
// as symbol.Actuate does. Its enumeration gets symbol.ActuatorEntries through intern.
func actuate(
	ctx context.Context,
	tx gorp.Tx,
	intern symbol.Intern,
	s v13.Schematic,
) error {
	for _, n := range s.Nodes {
		cfg, ok := n.Config.Variant.(v13.CustomActuatorNodeConfig)
		if !ok {
			continue
		}
		key, err := uuid.Parse(cfg.SpecKey)
		if err != nil {
			continue
		}
		sym, found, err := retrieveSymbol(ctx, tx, key)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		sym.Data = symbol.Actuate(sym.Data)
		if err = linkActuatorEntries(ctx, tx, intern, &sym); err != nil {
			return err
		}
		if err = gorp.NewCreate[symbol.Key, symbol.Symbol]().
			Entry(&sym).
			Exec(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

// linkActuatorEntries gives the state "active" of sym symbol.ActuatorEntries, when its
// enumeration lacks the entry On. It adds the entry to the enumeration when no other
// symbol links it, and links a new enumeration through intern when one does.
func linkActuatorEntries(
	ctx context.Context,
	tx gorp.Tx,
	intern symbol.Intern,
	sym *symbol.Symbol,
) error {
	for i, s := range sym.Data.States {
		state, ok := s.Variant.(symbol.EnumerationState)
		if !ok || state.Key != activeState || state.Enumeration == nil {
			continue
		}
		e, err := retrieveEnumeration(ctx, tx, *state.Enumeration)
		if err != nil {
			return err
		}
		entries, added := symbol.ActuatorEntries(e.Entries)
		if !added {
			return nil
		}
		shared, err := linkedElsewhere(ctx, tx, sym.Key, e.Key)
		if err != nil {
			return err
		}
		if !shared {
			e.Entries = entries
			return gorp.NewCreate[enumeration.Key, enumeration.Enumeration]().
				Entry(&e).
				Exec(ctx, tx)
		}
		key, err := intern(ctx, sym.Name, entries)
		if err != nil {
			return err
		}
		state.Enumeration = &key
		sym.Data.States[i] = symbol.State{Variant: state}
	}
	return nil
}

// linkedElsewhere returns whether a symbol other than the one with key links e.
func linkedElsewhere(
	ctx context.Context,
	tx gorp.Tx,
	key symbol.Key,
	e enumeration.Key,
) (bool, error) {
	return gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
		Where(gorp.Match(func(_ gorp.Context, s *symbol.Symbol) (bool, error) {
			return s.Key != key && slices.ContainsFunc(
				s.Data.States,
				func(st symbol.State) bool {
					es, ok := st.Variant.(symbol.EnumerationState)
					return ok && es.Enumeration != nil && *es.Enumeration == e
				},
			), nil
		})).
		Exists(ctx, tx)
}

func retrieveEnumeration(
	ctx context.Context,
	tx gorp.Tx,
	key enumeration.Key,
) (enumeration.Enumeration, error) {
	var e enumeration.Enumeration
	if err := gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
		Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](key)).
		Entry(&e).
		Exec(ctx, tx); err != nil {
		return enumeration.Enumeration{}, err
	}
	return e, nil
}

// dropUndrawnLooks gives each stored symbol with no commands its symbol.DrawnLooks.
func dropUndrawnLooks(ctx context.Context, tx gorp.Tx) error {
	var syms []symbol.Symbol
	if err := gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
		Entries(&syms).
		Exec(ctx, tx); err != nil {
		return err
	}
	static := make([]symbol.Symbol, 0, len(syms))
	for _, sym := range syms {
		if len(sym.Data.Commands) == 0 {
			sym.Data.Looks = symbol.DrawnLooks(sym.Data)
			static = append(static, sym)
		}
	}
	return gorp.NewCreate[symbol.Key, symbol.Symbol]().Entries(&static).Exec(ctx, tx)
}

// resolveSymbols gives each symbol node of s whose custom symbol the store does not
// hold a missing ref, and drops each state binding that names no state of its symbol.
func resolveSymbols(ctx context.Context, tx gorp.Tx, s *Schematic) error {
	for i, n := range s.Nodes {
		cfg, ok := n.Config.Variant.(SymbolNodeConfig)
		if !ok {
			continue
		}
		ref, ok := cfg.Spec.Variant.(symbol.CustomRef)
		if !ok {
			continue
		}
		sym, found, err := retrieveSymbol(ctx, tx, ref.Key)
		if err != nil {
			return err
		}
		if !found {
			cfg.Spec = symbol.Ref{Variant: symbol.MissingRef{Key: ref.Key.String()}}
		} else {
			cfg.States = slices.DeleteFunc(cfg.States, func(b StateBinding) bool {
				return !slices.ContainsFunc(
					sym.Data.States,
					func(st symbol.State) bool {
						return stateKey(st) == b.Key
					},
				)
			})
		}
		s.Nodes[i].Config.Variant = cfg
	}
	return nil
}

// retrieveSymbol returns the stored symbol with key, or false when the store does not
// hold it.
func retrieveSymbol(
	ctx context.Context,
	tx gorp.Tx,
	key symbol.Key,
) (symbol.Symbol, bool, error) {
	var sym symbol.Symbol
	err := gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
		Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](key)).
		Entry(&sym).
		Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return symbol.Symbol{}, false, nil
	}
	if err != nil {
		return symbol.Symbol{}, false, err
	}
	return sym, true, nil
}

func stateKey(st symbol.State) string {
	switch v := st.Variant.(type) {
	case symbol.EnumerationState:
		return v.Key
	case symbol.NumberState:
		return v.Key
	}
	return ""
}

// warnEmptyThresholds logs each light of s whose threshold is empty. Such a light never
// turned on, and its symbol node turns on as the built-in does.
func warnEmptyThresholds(s v13.Schematic, ins alamos.Instrumentation) {
	for _, n := range s.Nodes {
		l, ok := n.Config.Variant.(v13.LightNodeConfig)
		if ok && l.OnColor == nil && l.Threshold != nil &&
			l.Threshold.Lower == l.Threshold.Upper {
			ins.L.Warn(
				"a light with an empty threshold now turns on at 0.9 to 1.1",
				zap.Stringer("schematic", s.Key),
				zap.String("schematic_name", s.Name),
				zap.String("node", n.Key),
			)
		}
	}
}
