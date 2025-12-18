// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type GpuReport, ZERO_GPU_REPORT } from "@/perf/analyzer/types";

const HIGH_GPU_AVG_THRESHOLD = 80;
const HIGH_GPU_PEAK_THRESHOLD = 95;

export interface GpuContext {
  startPercent: number | null;
  endPercent: number | null;
  avgPercent: number | null;
  peakPercent: number | null;
}

/**
 * Analyzes GPU usage during performance tests.
 * Detects high GPU when average >80% or peak >95%.
 */
export class GpuAnalyzer {
  analyze(ctx: GpuContext): GpuReport {
    const detected =
      (ctx.avgPercent != null && ctx.avgPercent > HIGH_GPU_AVG_THRESHOLD) ||
      (ctx.peakPercent != null && ctx.peakPercent > HIGH_GPU_PEAK_THRESHOLD);

    return {
      ...ZERO_GPU_REPORT,
      detected,
      avgPercent: ctx.avgPercent != null ? math.roundTo(ctx.avgPercent) : null,
      peakPercent: ctx.peakPercent != null ? math.roundTo(ctx.peakPercent) : null,
      startPercent: ctx.startPercent != null ? math.roundTo(ctx.startPercent) : null,
      endPercent: ctx.endPercent != null ? math.roundTo(ctx.endPercent) : null,
    };
  }
}
