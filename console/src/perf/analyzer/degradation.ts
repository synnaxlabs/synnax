// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { type DegradationReport, ZERO_DEGRADATION_REPORT } from "@/perf/analyzer/types";

const FPS_DEGRADATION_THRESHOLD_PERCENT = 15;

export interface FPSContext {
  startFPS: number | null;
  endFPS: number | null;
}

/**
 * Analyzes FPS degradation by comparing start vs end frame rates.
 * Detects degradation when FPS drops more than 15% during the test.
 */
export class DegradationDetector {
  analyze(fps: FPSContext): DegradationReport {
    if (fps.startFPS == null || fps.endFPS == null) 
      return ZERO_DEGRADATION_REPORT;
    

    const fpsDrop =
      fps.startFPS > 0 ? ((fps.startFPS - fps.endFPS) / fps.startFPS) * 100 : 0;
    const detected = fpsDrop > FPS_DEGRADATION_THRESHOLD_PERCENT;

    return {
      ...ZERO_DEGRADATION_REPORT,
      detected,
      averageFrameRateStart: math.roundTo(fps.startFPS),
      averageFrameRateEnd: math.roundTo(fps.endFPS),
      frameRateDegradationPercent: math.roundTo(fpsDrop, 2),
    };
  }
}
