// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Mover is a part that the loop moves. */
export interface Mover {
  /**
   * Draws the part at the frame time now, in milliseconds.
   * @returns whether the part keeps moving. A part that stops leaves the loop.
   */
  step: (now: number) => boolean;
}

/**
 * Loop moves every moving part on one requestAnimationFrame loop. It asks for frames
 * only while a part moves.
 */
export class Loop {
  private readonly movers = new Set<Mover>();
  private handle: number | null = null;

  add(mover: Mover): void {
    this.movers.add(mover);
    this.handle ??= requestAnimationFrame(this.frame);
  }

  delete(mover: Mover): void {
    this.movers.delete(mover);
    if (this.movers.size > 0 || this.handle == null) return;
    cancelAnimationFrame(this.handle);
    this.handle = null;
  }

  private readonly frame = (now: number): void => {
    this.handle = null;
    for (const mover of this.movers) if (!mover.step(now)) this.movers.delete(mover);
    if (this.movers.size > 0) this.handle = requestAnimationFrame(this.frame);
  };
}
