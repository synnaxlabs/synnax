// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import type { Names } from "@/schematic/node/builtIns";
import { SYMBOL as ANGLED } from "@/schematic/node/valves/Angled";
import { SYMBOL as ANGLED_RELIEF } from "@/schematic/node/valves/AngledRelief";
import { SYMBOL as ANGLED_SPRING_LOADED_RELIEF } from "@/schematic/node/valves/AngledSpringLoadedRelief";
import { SYMBOL as BALL } from "@/schematic/node/valves/Ball";
import { SYMBOL as BREATHER } from "@/schematic/node/valves/Breather";
import { SYMBOL as BUTTERFLY_ONE } from "@/schematic/node/valves/ButterflyOne";
import { SYMBOL as BUTTERFLY_TWO } from "@/schematic/node/valves/ButterflyTwo";
import { SYMBOL as CHECK } from "@/schematic/node/valves/Check";
import { SYMBOL as CHECK_WITH_ARROW } from "@/schematic/node/valves/CheckWithArrow";
import { SYMBOL as ELECTRIC_REGULATOR } from "@/schematic/node/valves/ElectricRegulator";
import { SYMBOL as ELECTRIC_REGULATOR_MOTORIZED } from "@/schematic/node/valves/ElectricRegulatorMotorized";
import { SYMBOL as FOUR_WAY } from "@/schematic/node/valves/FourWay";
import { SYMBOL as GATE } from "@/schematic/node/valves/Gate";
import { SYMBOL as ISO_CHECK } from "@/schematic/node/valves/IsoCheck";
import { SYMBOL as MANUAL } from "@/schematic/node/valves/Manual";
import { SYMBOL as NEEDLE } from "@/schematic/node/valves/Needle";
import { SYMBOL as REGULATOR } from "@/schematic/node/valves/Regulator";
import { SYMBOL as REGULATOR_MANUAL } from "@/schematic/node/valves/RegulatorManual";
import { SYMBOL as RELIEF } from "@/schematic/node/valves/Relief";
import {
  NORMALLY_OPEN_SYMBOL,
  SYMBOL as SOLENOID,
} from "@/schematic/node/valves/Solenoid";
import { SYMBOL as SPRING_LOADED_RELIEF } from "@/schematic/node/valves/SpringLoadedRelief";
import { SYMBOL as THREE_WAY } from "@/schematic/node/valves/ThreeWay";
import { SYMBOL as THREE_WAY_BALL } from "@/schematic/node/valves/ThreeWayBall";
import { SYMBOL as VALVE } from "@/schematic/node/valves/Valve";

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  angled_relief_valve: ANGLED_RELIEF,
  angled_spring_loaded_relief_valve: ANGLED_SPRING_LOADED_RELIEF,
  angled_valve: ANGLED,
  ball_valve: BALL,
  breather_valve: BREATHER,
  butterfly_valve_one: BUTTERFLY_ONE,
  butterfly_valve_two: BUTTERFLY_TWO,
  check_valve: CHECK,
  check_valve_with_arrow: CHECK_WITH_ARROW,
  electric_regulator: ELECTRIC_REGULATOR,
  electric_regulator_motorized: ELECTRIC_REGULATOR_MOTORIZED,
  four_way_valve: FOUR_WAY,
  gate_valve: GATE,
  iso_check_valve: ISO_CHECK,
  manual_valve: MANUAL,
  needle_valve: NEEDLE,
  normally_open_solenoid_valve: NORMALLY_OPEN_SYMBOL,
  regulator: REGULATOR,
  regulator_manual: REGULATOR_MANUAL,
  relief_valve: RELIEF,
  solenoid_valve: SOLENOID,
  spring_loaded_relief_valve: SPRING_LOADED_RELIEF,
  three_way_ball_valve: THREE_WAY_BALL,
  three_way_valve: THREE_WAY,
  valve: VALVE,
} as const satisfies Record<string, schematic.symbol.Spec>;

/** NAMES maps the name of each built-in symbol to its names. */
export const NAMES = {
  angled_relief_valve: { name: "Angled relief", label: "Angled relief valve" },
  angled_spring_loaded_relief_valve: {
    name: "Angled spring loaded relief",
    label: "Angled spring loaded relief valve",
  },
  angled_valve: { name: "Angled", label: "Angled valve" },
  ball_valve: { name: "Ball", label: "Ball valve" },
  breather_valve: { name: "Breather", label: "Breather valve" },
  butterfly_valve_one: {
    name: "Butterfly (remote)",
    label: "Butterfly valve (remote)",
  },
  butterfly_valve_two: {
    name: "Butterfly (manual)",
    label: "Butterfly valve (manual)",
  },
  check_valve: { name: "Check", label: "Check valve" },
  check_valve_with_arrow: { name: "Check (arrow)", label: "Check valve" },
  electric_regulator: { name: "Electric", label: "Electric regulator" },
  electric_regulator_motorized: {
    name: "Motorized",
    label: "Electric regulator motorized",
  },
  four_way_valve: { name: "Four way", label: "Four way valve" },
  gate_valve: { name: "Gate", label: "Gate valve" },
  iso_check_valve: { name: "ISO check", label: "ISO check valve" },
  manual_valve: { name: "Manual", label: "Manual valve" },
  needle_valve: { name: "Needle", label: "Needle valve" },
  normally_open_solenoid_valve: {
    name: "Normally open solenoid",
    label: "Normally open solenoid",
  },
  regulator: { name: "Regulator", label: "Regulator" },
  regulator_manual: { name: "Manual", label: "Manual regulator" },
  relief_valve: { name: "Relief", label: "Relief valve" },
  solenoid_valve: { name: "Solenoid", label: "Solenoid" },
  spring_loaded_relief_valve: {
    name: "Spring loaded relief",
    label: "Spring loaded relief valve",
  },
  three_way_ball_valve: { name: "Three-way ball", label: "Three-way ball valve" },
  three_way_valve: { name: "Three way", label: "Three way" },
  valve: { name: "Generic", label: "Valve" },
} as const satisfies Record<keyof typeof BUILT_INS, Names>;
