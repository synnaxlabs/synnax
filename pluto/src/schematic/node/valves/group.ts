// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@/icon";
import { type Group } from "@/schematic/node/group";

export const GROUP: Group = {
  key: "valves",
  name: "Valves",
  Icon: Icon.Valve,
  symbols: [
    "valve",
    "solenoid_valve",
    "three_way_valve",
    "four_way_valve",
    "angled_valve",
    "ball_valve",
    "three_way_ball_valve",
    "gate_valve",
    "butterfly_valve_one",
    "butterfly_valve_two",
    "breather_valve",
    "manual_valve",
    "needle_valve",
    "relief_valve",
    "angled_relief_valve",
    "spring_loaded_relief_valve",
    "angled_spring_loaded_relief_valve",
    "check_valve",
    "iso_check_valve",
    "check_valve_with_arrow",
    "regulator",
    "regulator_manual",
    "electric_regulator",
    "electric_regulator_motorized",
  ],
};
