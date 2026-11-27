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
import svg from "@/schematic/node/fittings/Nozzle.svg?raw";

export const SYMBOL: schematic.symbol.Spec = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "1", position: { x: 0.1875, y: 0.1746 }, orientation: "left" },
    { key: "2", position: { x: 0.5, y: 0.015873 }, orientation: "top" },
    { key: "3", position: { x: 0.8125, y: 0.1746 }, orientation: "right" },
  ],
});
