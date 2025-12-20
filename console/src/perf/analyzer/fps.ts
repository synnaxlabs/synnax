// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type FpsReport, ZERO_FPS_REPORT } from "@/perf/analyzer/types";

const FPS_CHANGE_THRESHOLD_PERCENT = 15;

export interface FpsContext {
  startFps: number | null;
  endFps: number | null;
}

/**
 * Analyzes FPS changes by comparing start vs end frame rates.
 * Detects degradation when FPS drops more than 15% during the test.
 */
export class FpsAnalyzer {
  analyze(ctx: FpsContext): FpsReport {
    if (ctx.startFps == null || ctx.endFps == null) return ZERO_FPS_REPORT;

    const fpsDrop =
      ctx.startFps > 0 ? ((ctx.startFps - ctx.endFps) / ctx.startFps) * 100 : 0;
    const detected = fpsDrop > FPS_CHANGE_THRESHOLD_PERCENT;

    return {
      ...ZERO_FPS_REPORT,
      detected,
      startFps: math.roundTo(ctx.startFps),
      endFps: math.roundTo(ctx.endFps),
      changePercent: math.roundTo(fpsDrop, 2),
    };
  }
}
