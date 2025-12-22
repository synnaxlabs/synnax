// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status } from "@/perf/types";

/** Get status based on threshold. Use inverted=true when lower values are worse (e.g., FPS). */
export const getThresholdStatus = (
  value: number | null,
  warningThreshold: number,
  errorThreshold: number,
  inverted = false,
): Status => {
  if (value == null) return undefined;
  const compare = inverted
    ? (v: number, t: number) => v < t
    : (v: number, t: number) => v > t;
  if (compare(value, errorThreshold)) return "error";
  if (compare(value, warningThreshold)) return "warning";
  return undefined;
};

export interface AvgPeakThresholds {
  avgWarn: number;
  avgError: number;
  peakWarn: number;
  peakError: number;
}

/**
 * Get status based on both avg and peak values against their respective thresholds.
 * Use inverted=true when lower values are worse (e.g., FPS).
 * Returns the worst status between avg and peak.
 */
export const getAvgPeakStatus = (
  avg: number | null,
  peak: number | null,
  thresholds: AvgPeakThresholds,
  inverted = false,
): Status => {
  const compare = inverted
    ? (v: number, t: number) => v < t
    : (v: number, t: number) => v > t;

  let status: Status = undefined;

  if (peak != null) {
    if (compare(peak, thresholds.peakError)) status = "error";
    else if (compare(peak, thresholds.peakWarn)) status = "warning";
  }

  if (avg != null && status !== "error") {
    if (compare(avg, thresholds.avgError)) status = "error";
    else if (status !== "warning" && compare(avg, thresholds.avgWarn))
      status = "warning";
  }

  return status;
};
