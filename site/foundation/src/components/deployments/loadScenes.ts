// Copyright 2026 Synnax Labs, Inc. Licensed under licenses/BSL.txt.

// Keep each deployment in its own chunk. The carousel only requests the selected
// scene; nearby scenes can be warmed once the visible carousel is idle.
export const loadScenes = {
  industrial: () => import("./PlantScene").then((module) => module.PlantScene),
  aerospace: () => import("./AerospaceScene").then((module) => module.AerospaceScene),
  quantum: () => import("./QuantumScene").then((module) => module.QuantumScene),
  energy: () => import("./EnergyScene").then((module) => module.EnergyScene),
  marine: () => import("./MarineScene").then((module) => module.MarineScene),
};
