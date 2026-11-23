// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { color } from "@synnaxlabs/x";

type Variant = schematic.ElementConfigType;
type ConfigOf<V extends Variant> = Extract<schematic.ElementConfig, { variant: V }>;

type ValueColorField<V extends Variant> = {
  [K in keyof ConfigOf<V>]-?: NonNullable<ConfigOf<V>[K]> extends color.Color
    ? K
    : never;
}[keyof ConfigOf<V>];

// Each variant with a color value field must appear and list each such field, so a
// renamed, retyped, or missing field fails the type check.
type ValueColorFields = {
  readonly [V in Variant as [ValueColorField<V>] extends [never] ? never : V]: {
    readonly [K in ValueColorField<V>]: true;
  };
};

/** Every stored color field, role or not, that Selection lists. */
export const COLOR_FIELDS = [
  "strokeColor",
  "fillColor",
  "textColor",
  "levelColor",
  "axisColor",
  "onColor",
] as const;

/**
 * The color fields of each variant that store a color value instead of a reference. A
 * named color cannot go in one.
 */
export const VALUE_COLOR_FIELDS: Partial<
  Record<string, Readonly<Record<string, true>>>
> = {
  cap: { strokeColor: true },
  filter: { strokeColor: true },
  flow_straightener: { strokeColor: true },
  heater_element: { strokeColor: true },
  iso_cap: { strokeColor: true },
  iso_filter: { strokeColor: true },
  nozzle: { strokeColor: true },
  orifice: { strokeColor: true },
  orifice_plate: { strokeColor: true },
  strainer: { strokeColor: true },
  strainer_cone: { strokeColor: true },
  thruster: { strokeColor: true },
  vent: { strokeColor: true },
  flowmeter_general: { strokeColor: true },
  flowmeter_electromagnetic: { strokeColor: true },
  flowmeter_variable_area: { strokeColor: true },
  flowmeter_coriolis: { strokeColor: true },
  flowmeter_nozzle: { strokeColor: true },
  flowmeter_venturi: { strokeColor: true },
  flowmeter_ring_piston: { strokeColor: true },
  flowmeter_positive_displacement: { strokeColor: true },
  flowmeter_turbine: { strokeColor: true },
  flowmeter_pulse: { strokeColor: true },
  flowmeter_float_sensor: { strokeColor: true },
  flowmeter_orifice: { strokeColor: true },
  light: { strokeColor: true, onColor: true },
  switch: { strokeColor: true },
  agitator: { strokeColor: true },
  cross_beam_agitator: { strokeColor: true },
  flat_blade_agitator: { strokeColor: true },
  heat_exchanger_general: { strokeColor: true },
  heat_exchanger_m: { strokeColor: true },
  heat_exchanger_straight_tube: { strokeColor: true },
  helical_agitator: { strokeColor: true },
  paddle_agitator: { strokeColor: true },
  propeller_agitator: { strokeColor: true },
  rotary_mixer: { strokeColor: true },
  static_mixer: { strokeColor: true },
  cavity_pump: { strokeColor: true },
  centrifugal_compressor: { strokeColor: true },
  compressor: { strokeColor: true },
  diaphragm_pump: { strokeColor: true },
  ejection_pump: { strokeColor: true },
  ejector_compressor: { strokeColor: true },
  liquid_ring_compressor: { strokeColor: true },
  piston_pump: { strokeColor: true },
  pump: { strokeColor: true },
  roller_vane_compressor: { strokeColor: true },
  screw_pump: { strokeColor: true },
  turbo_compressor: { strokeColor: true },
  vacuum_pump: { strokeColor: true },
  burst_disc: { strokeColor: true },
  flame_arrestor: { strokeColor: true },
  flame_arrestor_detonation: { strokeColor: true },
  flame_arrestor_explosion: { strokeColor: true },
  flame_arrestor_fire_res: { strokeColor: true },
  flame_arrestor_fire_res_detonation: { strokeColor: true },
  iso_burst_disc: { strokeColor: true },
  angled_valve: { strokeColor: true },
  angled_relief_valve: { strokeColor: true },
  angled_spring_loaded_relief_valve: { strokeColor: true },
  ball_valve: { strokeColor: true },
  breather_valve: { strokeColor: true },
  butterfly_valve_one: { strokeColor: true },
  butterfly_valve_two: { strokeColor: true },
  check_valve: { strokeColor: true },
  check_valve_with_arrow: { strokeColor: true },
  electric_regulator: { strokeColor: true },
  electric_regulator_motorized: { strokeColor: true },
  four_way_valve: { strokeColor: true },
  gate_valve: { strokeColor: true },
  iso_check_valve: { strokeColor: true },
  manual_valve: { strokeColor: true },
  needle_valve: { strokeColor: true },
  regulator: { strokeColor: true },
  regulator_manual: { strokeColor: true },
  relief_valve: { strokeColor: true },
  solenoid_valve: { strokeColor: true },
  spring_loaded_relief_valve: { strokeColor: true },
  three_way_valve: { strokeColor: true },
  three_way_ball_valve: { strokeColor: true },
  valve: { strokeColor: true },
  cross_junction: { strokeColor: true },
  t_junction: { strokeColor: true },
} satisfies ValueColorFields;
