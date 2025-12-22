// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type GpuReport, type Severity, ZERO_GPU_REPORT } from "@/perf/analyzer/types";
import { THRESHOLDS } from "@/perf/constants";

export interface GpuContext {
  startPercent: number | null;
  endPercent: number | null;
  avgPercent: number | null;
  maxPercent: number | null;
}

/**
 * Analyzes GPU usage during performance tests.
 */
export class GpuAnalyzer {
  analyze(ctx: GpuContext): GpuReport {
    const effectiveMax = Math.max(ctx.maxPercent ?? 0, ctx.endPercent ?? 0);

    // Peak severity: based on maximum GPU usage
    let peakSeverity: Severity = "none";
    if (effectiveMax > THRESHOLDS.gpu.error) peakSeverity = "error";
    else if (effectiveMax > THRESHOLDS.gpu.warn) peakSeverity = "warning";

    // Avg severity: based on average GPU usage
    let avgSeverity: Severity = "none";
    if (ctx.avgPercent != null) 
      if (ctx.avgPercent > THRESHOLDS.gpuAvg.error) avgSeverity = "error";
      else if (ctx.avgPercent > THRESHOLDS.gpuAvg.warn) avgSeverity = "warning";
    

    return {
      ...ZERO_GPU_REPORT,
      peakSeverity,
      avgSeverity,
      avgPercent: ctx.avgPercent != null ? math.roundTo(ctx.avgPercent) : null,
      maxPercent: ctx.maxPercent != null ? math.roundTo(ctx.maxPercent) : null,
      startPercent: ctx.startPercent != null ? math.roundTo(ctx.startPercent) : null,
      endPercent: ctx.endPercent != null ? math.roundTo(ctx.endPercent) : null,
    };
  }
}
