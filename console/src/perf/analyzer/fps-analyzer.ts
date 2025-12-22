// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type FpsReport, type Severity, ZERO_FPS_REPORT } from "@/perf/analyzer/types";
import { THRESHOLDS } from "@/perf/constants";

export interface FpsContext {
  startFps: number | null;
  endFps: number | null;
}

export interface FpsAnalysisContext extends FpsContext {
  minFps: number | null;
  avgFps: number | null;
}

/**
 * Analyzes FPS during performance tests.
 * Severity is determined by absolute FPS values (min/avg) against thresholds.
 */
export class FpsAnalyzer {
  analyze(ctx: FpsAnalysisContext): FpsReport {
    const effectiveMin = Math.min(ctx.minFps ?? Infinity, ctx.endFps ?? Infinity);

    let fpsChange = 0;
    if (ctx.startFps != null && ctx.endFps != null && ctx.startFps > 0)
      fpsChange = ((ctx.startFps - ctx.endFps) / ctx.startFps) * 100;

    // Peak severity: based on minimum FPS (inverted - lower is worse)
    let peakSeverity: Severity = "none";
    if (effectiveMin < THRESHOLDS.fps.error) peakSeverity = "error";
    else if (effectiveMin < THRESHOLDS.fps.warn) peakSeverity = "warning";

    // Avg severity: based on average FPS (inverted - lower is worse)
    let avgSeverity: Severity = "none";
    if (ctx.avgFps != null) {
      if (ctx.avgFps < THRESHOLDS.fpsAvg.error) avgSeverity = "error";
      else if (ctx.avgFps < THRESHOLDS.fpsAvg.warn) avgSeverity = "warning";
    }

    return {
      ...ZERO_FPS_REPORT,
      peakSeverity,
      avgSeverity,
      startFps: ctx.startFps != null ? math.roundTo(ctx.startFps) : 0,
      endFps: ctx.endFps != null ? math.roundTo(ctx.endFps) : 0,
      changePercent: math.roundTo(fpsChange, 2),
    };
  }
}
