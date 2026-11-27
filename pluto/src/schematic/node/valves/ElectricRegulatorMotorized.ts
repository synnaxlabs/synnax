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
import svg from "@/schematic/node/valves/ElectricRegulatorMotorized.svg?raw";

export const SYMBOL: schematic.symbol.Spec = Custom.createBuiltIn({
  svg,
  handles: [
    { key: "2", position: { x: 0.356321, y: 0.1372549 }, orientation: "left" },
    { key: "4", position: { x: 0.5, y: 0.0196078 }, orientation: "top" },
    { key: "3", position: { x: 0.643679, y: 0.1372549 }, orientation: "right" },
    { key: "1", position: { x: 0.241379, y: 0.46078 }, orientation: "left" },
    { key: "5", position: { x: 0.758621, y: 0.46078 }, orientation: "right" },
    { key: "6", position: { x: 0.022989, y: 0.7843137 }, orientation: "left" },
    { key: "7", position: { x: 0.977011, y: 0.7843137 }, orientation: "right" },
  ],
});
