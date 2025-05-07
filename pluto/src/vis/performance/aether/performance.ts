// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/ether";

export const performanceStateZ = z.object({
  fps: z.number(),
});

export class Performance extends aether.Leaf<typeof performanceStateZ> {
  static readonly TYPE = "performance";
  schema = performanceStateZ;
  interval: ReturnType<typeof setInterval> | null = null;

  async afterUpdate(): Promise<void> {
    // Convert duration to FPS (1000ms / avgDuration)
    if (this.interval != null) clearInterval(this.interval);
    this.interval = setInterval(() => {
      const entries = performance.getEntriesByName(
        "alamos.trace.duration.render-cycle",
      );
      if (entries.length === 0) return;

      // Calculate average duration in milliseconds
      const avgDuration =
        entries.reduce((acc, entry) => acc + entry.duration, 0) / entries.length;

      const fps = Math.round(1000 / avgDuration);
      performance.clearMarks();
      performance.clearMeasures();
      this.setState((p) => ({ ...p, fps }));
    }, 1000);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Performance.TYPE]: Performance,
};
