// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type color } from "@synnaxlabs/x";

type Config = schematic.ElementConfig;
type Variant = Config["variant"];
type ConfigOf<V extends Variant> = Extract<Config, { variant: V }>;

type ValueColorField<V extends Variant> = {
  [K in keyof ConfigOf<V>]-?: NonNullable<ConfigOf<V>[K]> extends color.Color
    ? K
    : never;
}[keyof ConfigOf<V>];

// Each variant with a color value field must appear, and each list holds only such
// fields, so a renamed or retyped field fails the type check.
type ValueColorFields = {
  readonly [
    V in Variant as [ValueColorField<V>] extends [never] ? never : V
  ]: readonly ValueColorField<V>[];
};

/** Every stored color field, role or not, that Selection lists. */
export const COLOR_FIELDS = [
  "strokeColor",
  "fillColor",
  "textColor",
  "levelColor",
  "axisColor",
  "onColor",
  "stalenessColor",
] as const;

const STROKE = ["strokeColor"] as const;
const STALENESS = ["stalenessColor"] as const;
const STROKE_AND_STALENESS = ["strokeColor", "stalenessColor"] as const;

const valueColorFields: ValueColorFields = {
  cap: STROKE,
  filter: STROKE,
  flow_straightener: STROKE,
  heater_element: STROKE,
  iso_cap: STROKE,
  iso_filter: STROKE,
  nozzle: STROKE,
  orifice: STROKE,
  orifice_plate: STROKE,
  strainer: STROKE,
  strainer_cone: STROKE,
  thruster: STROKE_AND_STALENESS,
  vent: STROKE,
  flowmeter_general: STROKE,
  flowmeter_electromagnetic: STROKE,
  flowmeter_variable_area: STROKE,
  flowmeter_coriolis: STROKE,
  flowmeter_nozzle: STROKE,
  flowmeter_venturi: STROKE,
  flowmeter_ring_piston: STROKE,
  flowmeter_positive_displacement: STROKE,
  flowmeter_turbine: STROKE,
  flowmeter_pulse: STROKE,
  flowmeter_float_sensor: STROKE,
  flowmeter_orifice: STROKE,
  gauge: STALENESS,
  input: STALENESS,
  light: ["strokeColor", "onColor", "stalenessColor"],
  select: STALENESS,
  scale: STALENESS,
  setpoint: STALENESS,
  state_indicator: STALENESS,
  string_display: STALENESS,
  switch: STROKE_AND_STALENESS,
  value: STALENESS,
  agitator: STROKE_AND_STALENESS,
  cross_beam_agitator: STROKE_AND_STALENESS,
  flat_blade_agitator: STROKE_AND_STALENESS,
  heat_exchanger_general: STROKE,
  heat_exchanger_m: STROKE,
  heat_exchanger_straight_tube: STROKE,
  helical_agitator: STROKE_AND_STALENESS,
  paddle_agitator: STROKE_AND_STALENESS,
  propeller_agitator: STROKE_AND_STALENESS,
  rotary_mixer: STROKE_AND_STALENESS,
  static_mixer: STROKE,
  cavity_pump: STROKE_AND_STALENESS,
  centrifugal_compressor: STROKE_AND_STALENESS,
  compressor: STROKE_AND_STALENESS,
  diaphragm_pump: STROKE_AND_STALENESS,
  ejection_pump: STROKE_AND_STALENESS,
  ejector_compressor: STROKE_AND_STALENESS,
  liquid_ring_compressor: STROKE_AND_STALENESS,
  piston_pump: STROKE_AND_STALENESS,
  pump: STROKE_AND_STALENESS,
  roller_vane_compressor: STROKE_AND_STALENESS,
  screw_pump: STROKE_AND_STALENESS,
  turbo_compressor: STROKE_AND_STALENESS,
  vacuum_pump: STROKE_AND_STALENESS,
  burst_disc: STROKE,
  flame_arrestor: STROKE,
  flame_arrestor_detonation: STROKE,
  flame_arrestor_explosion: STROKE,
  flame_arrestor_fire_res: STROKE,
  flame_arrestor_fire_res_detonation: STROKE,
  iso_burst_disc: STROKE,
  angled_valve: STROKE_AND_STALENESS,
  angled_relief_valve: STROKE,
  angled_spring_loaded_relief_valve: STROKE,
  ball_valve: STROKE_AND_STALENESS,
  breather_valve: STROKE,
  butterfly_valve_one: STROKE_AND_STALENESS,
  butterfly_valve_two: STROKE_AND_STALENESS,
  check_valve: STROKE,
  check_valve_with_arrow: STROKE,
  electric_regulator: STROKE,
  electric_regulator_motorized: STROKE,
  four_way_valve: STROKE_AND_STALENESS,
  gate_valve: STROKE_AND_STALENESS,
  iso_check_valve: STROKE,
  manual_valve: STROKE,
  needle_valve: STROKE,
  regulator: STROKE,
  regulator_manual: STROKE,
  relief_valve: STROKE,
  solenoid_valve: STROKE_AND_STALENESS,
  spring_loaded_relief_valve: STROKE,
  three_way_valve: STROKE_AND_STALENESS,
  three_way_ball_valve: STROKE_AND_STALENESS,
  valve: STROKE_AND_STALENESS,
  cross_junction: STROKE,
  tank: STALENESS,
  t_junction: STROKE,
  custom_actuator: STALENESS,
};

/**
 * The color fields of each variant that store a color value instead of a reference. A
 * named color cannot go in one.
 */
export const VALUE_COLOR_FIELDS: Partial<Record<string, readonly string[]>> =
  valueColorFields;
