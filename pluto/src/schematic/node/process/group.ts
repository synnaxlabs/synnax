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
  key: "process",
  name: "Process",
  Icon: Icon.Process,
  symbols: [
    "heat_exchanger_general",
    "heat_exchanger_m",
    "heat_exchanger_straight_tube",
    "static_mixer",
    "rotary_mixer",
    "agitator",
    "propeller_agitator",
    "flat_blade_agitator",
    "paddle_agitator",
    "cross_beam_agitator",
    "helical_agitator",
  ],
};
