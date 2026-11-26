// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { Custom } from "@/schematic/node/common/custom";
import svg from "@/schematic/node/general/switch/Switch.svg?raw";

const ON: schematic.symbol.Condition = { type: "equals", state: "active", values: [1] };

// The track fills while on, and the far knob takes the place of the near one.
export const SYMBOL: schematic.symbol.Spec = Custom.createBuiltIn({
  family: "switch",
  svg,
  handles: [],
  parts: [
    { key: "track", name: "Track", selectors: [".pluto-switch-symbol__track"] },
    { key: "knob-off", name: "Knob off", selectors: [".pluto-switch-symbol__knob"] },
    { key: "knob-on", name: "Knob on", selectors: [".pluto-switch-symbol__knob-on"] },
  ],
  looks: [
    Custom.STROKE_LOOK,
    Custom.createFillLook(1, "track"),
    { key: "knob-off", part: "knob-off", when: ON, hidden: true },
    { key: "knob-on", part: "knob-on", hidden: true },
    {
      key: "knob-on-shown",
      part: "knob-on",
      when: ON,
      hidden: false,
      fill: { type: "contrast", state: "active" },
    },
  ],
});
