// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type CpuReport, type Severity, ZERO_CPU_REPORT } from "@/perf/analyzer/types";
import { THRESHOLDS } from "@/perf/constants";

export interface CpuContext {
  startPercent: number | null;
  endPercent: number | null;
  avgPercent: number | null;
  maxPercent: number | null;
}

/**
 * Analyzes CPU usage during performance tests.
 * Severity is determined by current sample's value and average and peak CPU percent.
 */
export class CpuAnalyzer {
  analyze(ctx: CpuContext): CpuReport {
    const effectiveMax = Math.max(ctx.maxPercent ?? 0, ctx.endPercent ?? 0);

    // Peak severity: based on maximum CPU usage
    let peakSeverity: Severity = "none";
    if (effectiveMax > THRESHOLDS.cpu.error) peakSeverity = "error";
    else if (effectiveMax > THRESHOLDS.cpu.warn) peakSeverity = "warning";

    // Avg severity: based on average CPU usage
    let avgSeverity: Severity = "none";
    if (ctx.avgPercent != null) 
      if (ctx.avgPercent > THRESHOLDS.cpuAvg.error) avgSeverity = "error";
      else if (ctx.avgPercent > THRESHOLDS.cpuAvg.warn) avgSeverity = "warning";
    

    return {
      ...ZERO_CPU_REPORT,
      peakSeverity,
      avgSeverity,
      avgPercent: ctx.avgPercent != null ? math.roundTo(ctx.avgPercent) : null,
      maxPercent: ctx.maxPercent != null ? math.roundTo(ctx.maxPercent) : null,
      startPercent: ctx.startPercent != null ? math.roundTo(ctx.startPercent) : null,
      endPercent: ctx.endPercent != null ? math.roundTo(ctx.endPercent) : null,
    };
  }
}
