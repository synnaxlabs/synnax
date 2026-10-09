// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Keep each deployment in its own chunk. The carousel only requests the selected
// scene; nearby scenes can be warmed once the visible carousel is idle.
export const loadScenes = {
  industrial: () => import("./PlantScene").then((module) => module.PlantScene),
  aerospace: () => import("./AerospaceScene").then((module) => module.AerospaceScene),
  quantum: () => import("./QuantumScene").then((module) => module.QuantumScene),
  energy: () => import("./EnergyScene").then((module) => module.EnergyScene),
  marine: () => import("./MarineScene").then((module) => module.MarineScene),
};
