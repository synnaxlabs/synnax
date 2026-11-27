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
import svg from "@/schematic/node/valves/Check.svg?raw";

export const SYMBOL: schematic.symbol.Spec = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "1", position: { x: 0.042222, y: 0.488372 }, orientation: "left" },
    { key: "2", position: { x: 0.942038, y: 0.488372 }, orientation: "right" },
  ],
});
