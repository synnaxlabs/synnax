// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan, TimeStamp } from "@/telem/telem";

/**
 * Calculates and tracks clock skew between two systems using a midpoint
 * synchronization algorithm. Useful for distributed systems where clock
 * synchronization is critical.
 */
export class ClockSkewCalculator {
  private readonly now: () => TimeStamp;
  private localStartT: TimeStamp = new TimeStamp(0);
  private lastSkew: TimeSpan = TimeSpan.ZERO;

  constructor(now: () => TimeStamp = () => TimeStamp.now()) {
    this.now = now;
  }

  start(): void {
    this.localStartT = this.now();
  }

  end(remoteMidpointT: TimeStamp): void {
    const localEndT = this.now();
    const halfSpan = localEndT.span(this.localStartT).valueOf() / 2n;
    const thisMidpointT = this.localStartT.add(halfSpan);
    this.lastSkew = new TimeSpan(
      thisMidpointT.valueOf() - remoteMidpointT.valueOf(),
    );
  }

  get skew(): TimeSpan {
    return this.lastSkew;
  }

  exceeds(threshold: CrudeTimeSpan): boolean {
    const skewVal = this.skew.valueOf();
    const abs = skewVal < 0n ? -skewVal : skewVal;
    return abs > new TimeSpan(threshold).valueOf();
  }
}
