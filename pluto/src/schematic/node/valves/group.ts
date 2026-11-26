// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import type { Group } from "@/schematic/node/group";

export const GROUP: Group = {
  key: "valves",
  scope: "schematic_symbol",
  name: "Valves",
  Icon: Icon.Valve,
  symbols: [
    "angled_valve",
    "angled_relief_valve",
    "angled_spring_loaded_relief_valve",
    "ball_valve",
    "breather_valve",
    "butterfly_valve_one",
    "butterfly_valve_two",
    "check_valve",
    "check_valve_with_arrow",
    "electric_regulator",
    "electric_regulator_motorized",
    "four_way_valve",
    "gate_valve",
    "iso_check_valve",
    "manual_valve",
    "needle_valve",
    "regulator",
    "regulator_manual",
    "relief_valve",
    "solenoid_valve",
    "spring_loaded_relief_valve",
    "three_way_valve",
    "three_way_ball_valve",
    "valve",
  ],
};
