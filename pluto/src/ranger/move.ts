// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type NumericTimeRange, TimeStamp } from "@synnaxlabs/x";

/** An open start or end. */
export const UNSET = TimeStamp.MAX.nanoseconds;

/**
 * Commits a start edit. A start moved past the end drags the end with it, keeping
 * the duration. Unscheduling clears the end too.
 */
export const moveStart = (
  { start, end }: NumericTimeRange,
  next: number,
): NumericTimeRange => {
  if (next >= UNSET) return { start: UNSET, end: UNSET };
  if (end < UNSET && next > end) return { start: next, end: next + (end - start) };
  return { start: next, end };
};

/**
 * Commits an end edit. An end moved before the start drags the start with it,
 * keeping the duration; an open range has none to keep, so the start clamps.
 */
export const moveEnd = (
  { start, end }: NumericTimeRange,
  next: number,
): NumericTimeRange => {
  if (next < start)
    return { start: end < UNSET ? next - (end - start) : next, end: next };
  return { start, end: next };
};
