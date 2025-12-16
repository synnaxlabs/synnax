// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Measures frame rate using requestAnimationFrame.
 * Calculates FPS by counting frames over a sliding window.
 */
export class FrameRateCollector {
  private frameCount = 0;
  private lastFPSUpdate = 0;
  private currentFPS = 60;
  private rafId: number | null = null;
  private running = false;

  /** Start measuring frame rate. */
  start(): void {
    if (this.running) return;
    this.running = true;
    this.frameCount = 0;
    this.lastFPSUpdate = performance.now();
    this.currentFPS = 60;
    this.measureFrame();
  }

  /** Stop measuring frame rate. */
  stop(): void {
    this.running = false;
    if (this.rafId != null) {
      cancelAnimationFrame(this.rafId);
      this.rafId = null;
    }
  }

  /** Get the current measured FPS. */
  getCurrentFPS(): number {
    return Math.round(this.currentFPS * 10) / 10;
  }

  private measureFrame = (): void => {
    if (!this.running) return;

    this.frameCount++;
    const now = performance.now();
    const elapsed = now - this.lastFPSUpdate;

    // Update FPS every second
    if (elapsed >= 1000) {
      this.currentFPS = (this.frameCount / elapsed) * 1000;
      this.frameCount = 0;
      this.lastFPSUpdate = now;
    }

    this.rafId = requestAnimationFrame(this.measureFrame);
  };
}
