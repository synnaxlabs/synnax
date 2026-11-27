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
import svg from "@/schematic/node/pumps/Turbo.svg?raw";

export const SYMBOL: schematic.symbol.Spec = Custom.createBuiltIn({
  family: "pump",
  svg,
  handles: [
    { key: "1", position: { x: 0.0455, y: 0.5 }, orientation: "left" },
    { key: "2", position: { x: 0.9545, y: 0.5 }, orientation: "right" },
    { key: "3", position: { x: 0.5, y: 0.0455 }, orientation: "top" },
    { key: "4", position: { x: 0.5, y: 0.9545 }, orientation: "bottom" },
  ],
});
