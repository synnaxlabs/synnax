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
  key: "meters",
  name: "Flowmeters",
  Icon: Icon.Rule,
  symbols: [
    "flowmeter_general",
    "flowmeter_electromagnetic",
    "flowmeter_variable_area",
    "flowmeter_coriolis",
    "flowmeter_nozzle",
    "flowmeter_venturi",
    "flowmeter_ring_piston",
    "flowmeter_positive_displacement",
    "flowmeter_turbine",
    "flowmeter_pulse",
    "flowmeter_float_sensor",
    "flowmeter_orifice",
  ],
};
