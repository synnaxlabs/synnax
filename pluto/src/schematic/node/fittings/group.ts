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
  key: "fittings",
  name: "Fittings",
  Icon: Icon.Fitting,
  symbols: [
    "cap",
    "iso_cap",
    "orifice",
    "orifice_plate",
    "vent",
    "nozzle",
    "heater_element",
    "thruster",
    "filter",
    "iso_filter",
    "strainer",
    "strainer_cone",
    "flow_straightener",
  ],
};
