// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/x/spatial"
)

const (
	activeState = "active"
	deactivate  = "deactivate"
	activate    = "activate"
)

// family holds the enumeration and command names that one kind of equipment shares.
type family struct {
	enumeration          enumeration.Key
	deactivate, activate string
}

func static(name string) Builtin {
	return Builtin{Name: name, States: []State{}, Commands: []Command{}}
}

func active(enum enumeration.Key) State {
	return State{Variant: EnumerationState{
		BaseState:   BaseState{Key: activeState, Name: "Active"},
		Enumeration: &enum,
		Match: Matcher{
			Variant: WithinMatcher{Bounds: spatial.Bounds{Lower: 0.9, Upper: 1.1}},
		},
	}}
}

func command(key, name string, value float64) Command {
	state := activeState
	return Command{
		Key:   key,
		Name:  name,
		State: &state,
		Value: CommandValue{Variant: FixedCommandValue{Value: value}},
	}
}

// toggle returns a built-in whose two commands write 0 and 1 to its active state.
func toggle(name string, f family) Builtin {
	return Builtin{
		Name:   name,
		States: []State{active(f.enumeration)},
		Commands: []Command{
			command(deactivate, f.deactivate, 0),
			command(activate, f.activate, 1),
		},
	}
}

// Builtins returns the symbols that the Core ships, by name, with their states and
// commands. Pluto holds their drawings.
func Builtins() []Builtin {
	var (
		valve             = family{enumeration.ValveKey, "Close", "Open"}
		normallyOpenValve = family{enumeration.NormallyOpenValveKey, "Open", "Close"}
		pump              = family{enumeration.PumpKey, "Stop", "Start"}
		switches          = family{enumeration.SwitchKey, "Turn off", "Turn on"}
		out               []Builtin
	)
	for _, name := range []string{
		"cap", "filter", "flow_straightener", "heater_element", "iso_cap", "iso_filter",
		"nozzle", "orifice", "orifice_plate", "strainer", "strainer_cone", "vent",
		"flowmeter_general", "flowmeter_electromagnetic", "flowmeter_variable_area",
		"flowmeter_coriolis", "flowmeter_nozzle", "flowmeter_venturi",
		"flowmeter_ring_piston", "flowmeter_positive_displacement", "flowmeter_turbine",
		"flowmeter_pulse", "flowmeter_float_sensor", "flowmeter_orifice",
		"heat_exchanger_general", "heat_exchanger_m", "heat_exchanger_straight_tube",
		"static_mixer", "burst_disc", "flame_arrestor", "flame_arrestor_detonation",
		"flame_arrestor_explosion", "flame_arrestor_fire_res",
		"flame_arrestor_fire_res_detonation", "iso_burst_disc", "check_valve",
		"check_valve_with_arrow", "electric_regulator", "electric_regulator_motorized",
		"iso_check_valve", "regulator", "regulator_manual", "cross_junction",
		"t_junction",
	} {
		out = append(out, static(name))
	}
	for _, group := range []struct {
		family
		names []string
	}{
		{valve, []string{
			"valve", "angled_valve", "ball_valve", "butterfly_valve_one",
			"butterfly_valve_two", "four_way_valve", "gate_valve", "three_way_valve",
			"three_way_ball_valve", "solenoid_valve", "angled_relief_valve",
			"angled_spring_loaded_relief_valve", "breather_valve", "manual_valve",
			"needle_valve", "relief_valve", "spring_loaded_relief_valve",
		}},
		{normallyOpenValve, []string{"normally_open_solenoid_valve"}},
		{pump, []string{
			"pump", "cavity_pump", "diaphragm_pump", "ejection_pump", "piston_pump",
			"screw_pump", "vacuum_pump", "compressor", "centrifugal_compressor",
			"ejector_compressor", "liquid_ring_compressor", "roller_vane_compressor",
			"turbo_compressor", "agitator", "cross_beam_agitator",
			"flat_blade_agitator", "helical_agitator", "paddle_agitator",
			"propeller_agitator", "rotary_mixer",
		}},
		{switches, []string{"switch", "thruster"}},
	} {
		for _, name := range group.names {
			out = append(out, toggle(name, group.family))
		}
	}
	return append(out, Builtin{
		Name:     "light",
		States:   []State{active(enumeration.LightKey)},
		Commands: []Command{},
	})
}
