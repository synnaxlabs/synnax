// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type CpuReport, ZERO_CPU_REPORT } from "@/perf/analyzer/types";

const HIGH_CPU_AVG_THRESHOLD = 50;
const HIGH_CPU_PEAK_THRESHOLD = 80;

export interface CpuContext {
  startPercent: number | null;
  endPercent: number | null;
  avgPercent: number | null;
  peakPercent: number | null;
}

/**
 * Analyzes CPU usage during performance tests.
 * Detects high CPU when average >50% or peak >80%.
 */
export class CpuAnalyzer {
  analyze(ctx: CpuContext): CpuReport {
    const detected =
      (ctx.avgPercent != null && ctx.avgPercent > HIGH_CPU_AVG_THRESHOLD) ||
      (ctx.peakPercent != null && ctx.peakPercent > HIGH_CPU_PEAK_THRESHOLD);

    return {
      ...ZERO_CPU_REPORT,
      detected,
      avgPercent: ctx.avgPercent != null ? math.roundTo(ctx.avgPercent) : null,
      peakPercent: ctx.peakPercent != null ? math.roundTo(ctx.peakPercent) : null,
      startPercent: ctx.startPercent != null ? math.roundTo(ctx.startPercent) : null,
      endPercent: ctx.endPercent != null ? math.roundTo(ctx.endPercent) : null,
    };
  }
}
