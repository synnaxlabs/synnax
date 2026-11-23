// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { Fittings } from "@/schematic/node/fittings";
import { Flowmeters } from "@/schematic/node/flowmeters";
import { General } from "@/schematic/node/general";
import { Process } from "@/schematic/node/process";
import { Pumps } from "@/schematic/node/pumps";
import { Safety } from "@/schematic/node/safety";
import { Valves } from "@/schematic/node/valves";
import { Vessels } from "@/schematic/node/vessels";

const SPECS = {
  ...Fittings.BUILT_INS,
  ...Flowmeters.BUILT_INS,
  ...General.BUILT_INS,
  ...Process.BUILT_INS,
  ...Pumps.BUILT_INS,
  ...Safety.BUILT_INS,
  ...Valves.BUILT_INS,
  ...Vessels.BUILT_INS,
};

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS: ReadonlyMap<string, schematic.symbol.Spec> = new Map(
  Object.entries(SPECS),
);

/**
 * BUILT_IN_NAMES maps the name of each built-in symbol to the name it shows in the
 * toolbar, which is also the label of a new node of it.
 */
export const BUILT_IN_NAMES: ReadonlyMap<string, string> = new Map(
  Object.entries({
    cap: "Cap",
    filter: "Filter",
    flow_straightener: "Flow straightener",
    heater_element: "Heater element",
    iso_cap: "ISO cap",
    iso_filter: "ISO filter",
    nozzle: "Nozzle",
    orifice: "Orifice",
    orifice_plate: "Orifice plate",
    strainer: "Strainer",
    strainer_cone: "Strainer cone",
    thruster: "Thruster",
    vent: "Vent",
    flowmeter_coriolis: "Coriolis flowmeter",
    flowmeter_electromagnetic: "Electromagnetic flowmeter",
    flowmeter_float_sensor: "Float sensor flowmeter",
    flowmeter_general: "General flowmeter",
    flowmeter_nozzle: "Nozzle flowmeter",
    flowmeter_orifice: "Orifice flowmeter",
    flowmeter_positive_displacement: "Positive displacement flowmeter",
    flowmeter_pulse: "Pulse flowmeter",
    flowmeter_ring_piston: "Ring piston flowmeter",
    flowmeter_turbine: "Turbine flowmeter",
    flowmeter_variable_area: "Variable area flowmeter",
    flowmeter_venturi: "Venturi flowmeter",
    light: "Light",
    switch: "Switch",
    agitator: "Agitator",
    cross_beam_agitator: "Cross beam agitator",
    flat_blade_agitator: "Flat blade agitator",
    heat_exchanger_general: "General heat exchanger",
    heat_exchanger_m: "M heat exchanger",
    heat_exchanger_straight_tube: "Straight tube heat exchanger",
    helical_agitator: "Helical agitator",
    paddle_agitator: "Paddle agitator",
    propeller_agitator: "Propeller agitator",
    rotary_mixer: "Rotary mixer",
    static_mixer: "Static mixer",
    cavity_pump: "Cavity pump",
    centrifugal_compressor: "Centrifugal compressor",
    compressor: "Compressor",
    diaphragm_pump: "Diaphragm pump",
    ejection_pump: "Ejection pump",
    ejector_compressor: "Ejector compressor",
    liquid_ring_compressor: "Liquid ring compressor",
    piston_pump: "Piston pump",
    pump: "Pump",
    roller_vane_compressor: "Roller vane compressor",
    screw_pump: "Screw pump",
    turbo_compressor: "Turbo compressor",
    vacuum_pump: "Vacuum pump",
    burst_disc: "Burst disc",
    flame_arrestor: "Flame arrestor",
    flame_arrestor_detonation: "Flame arrestor (detonation-proof)",
    flame_arrestor_explosion: "Flame arrestor (explosion-proof)",
    flame_arrestor_fire_res: "Flame arrestor (fire resistant)",
    flame_arrestor_fire_res_detonation:
      "Flame arrestor (fire resistant and detonation-proof)",
    iso_burst_disc: "ISO burst disc",
    angled_relief_valve: "Angled relief valve",
    angled_spring_loaded_relief_valve: "Angled spring loaded relief valve",
    angled_valve: "Angled valve",
    ball_valve: "Ball valve",
    breather_valve: "Breather valve",
    butterfly_valve_one: "Butterfly valve (remote)",
    butterfly_valve_two: "Butterfly valve (manual)",
    check_valve: "Check valve",
    check_valve_with_arrow: "Check valve",
    electric_regulator: "Electric regulator",
    electric_regulator_motorized: "Electric regulator motorized",
    four_way_valve: "Four way valve",
    gate_valve: "Gate valve",
    iso_check_valve: "ISO check valve",
    manual_valve: "Manual valve",
    needle_valve: "Needle valve",
    normally_open_solenoid_valve: "Normally open solenoid",
    regulator: "Regulator",
    regulator_manual: "Manual regulator",
    relief_valve: "Relief valve",
    solenoid_valve: "Solenoid",
    spring_loaded_relief_valve: "Spring loaded relief valve",
    three_way_ball_valve: "Three-way ball valve",
    three_way_valve: "Three way",
    valve: "Valve",
    cross_junction: "",
    t_junction: "",
  } satisfies Record<keyof typeof SPECS, string>),
);
