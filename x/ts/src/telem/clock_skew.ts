// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@/math";
import { TimeSpan, TimeStamp } from "@/telem/telem";

/**
 * Calculates and tracks clock skew between two systems using a midpoint
 * synchronization algorithm. This is useful for distributed systems where
 * clock synchronization is critical.
 */
export class ClockSkewCalculator {
  now: () => TimeStamp;
  localStartT: TimeStamp = TimeStamp.ZERO;
  accumulatedSkew: bigint = 0n;
  n: number = 0;

  constructor(now: () => TimeStamp = TimeStamp.now.bind(TimeSpan)) {
    this.now = now;
  }

  /** Starts a new clock skew measurement. */
  start(): void {
    this.localStartT = this.now();
  }

  /**
   * Completes a clock skew measurement.
   *
   * Uses the midpoint method: local_midpoint = start + (end - start) / 2.
   * The skew is then: local_midpoint - remote_midpoint.
   */
  end(remoteMidpoint: TimeStamp): void {
    const localEnd = this.now();
    const halfTrip = (localEnd.valueOf() - this.localStartT.valueOf()) / 2n;
    const localMid = this.localStartT.valueOf() + halfTrip;
    const skew = localMid - remoteMidpoint.valueOf();
    this.accumulatedSkew += skew;
    this.n++;
  }

  /** Returns the average clock skew across all measurements. */
  skew(): TimeSpan {
    if (this.n === 0) return TimeSpan.ZERO;
    return new TimeSpan(this.accumulatedSkew / BigInt(this.n));
  }

  /**
   * Checks if the absolute value of the average clock skew exceeds a threshold.
   */
  exceeds(threshold: TimeSpan): boolean {
    return new TimeSpan(math.abs(this.skew().valueOf())).greaterThan(threshold);
  }
}
