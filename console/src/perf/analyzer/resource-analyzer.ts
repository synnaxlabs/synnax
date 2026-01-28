// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type Severity } from "@/perf/analyzer/types";

export interface ResourceContext {
  startPercent: number | null;
  endPercent: number | null;
  avgPercent: number | null;
  maxPercent: number | null;
}

export interface ResourceReport {
  peakSeverity: Severity;
  avgSeverity: Severity;
  avgPercent: number | null;
  maxPercent: number | null;
  startPercent: number | null;
  endPercent: number | null;
}

interface Thresholds {
  warn: number;
  error: number;
}

export class ResourceAnalyzer {
  constructor(
    private readonly peakThresholds: Thresholds,
    private readonly avgThresholds: Thresholds,
  ) {}

  analyze(ctx: ResourceContext): ResourceReport {
    const effectiveMax = Math.max(ctx.maxPercent ?? 0, ctx.endPercent ?? 0);

    let peakSeverity: Severity = "none";
    if (effectiveMax > this.peakThresholds.error) peakSeverity = "error";
    else if (effectiveMax > this.peakThresholds.warn) peakSeverity = "warning";

    let avgSeverity: Severity = "none";
    if (ctx.avgPercent != null)
      if (ctx.avgPercent > this.avgThresholds.error) avgSeverity = "error";
      else if (ctx.avgPercent > this.avgThresholds.warn) avgSeverity = "warning";

    return {
      peakSeverity,
      avgSeverity,
      avgPercent: ctx.avgPercent != null ? math.roundTo(ctx.avgPercent) : null,
      maxPercent: ctx.maxPercent != null ? math.roundTo(ctx.maxPercent) : null,
      startPercent: ctx.startPercent != null ? math.roundTo(ctx.startPercent) : null,
      endPercent: ctx.endPercent != null ? math.roundTo(ctx.endPercent) : null,
    };
  }
}
