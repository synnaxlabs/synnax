// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

interface LongTaskEntry {
  timestamp: number;
  duration: number;
}

/**
 * Tracks long tasks (>50ms) that block the main thread.
 * Uses the PerformanceObserver API with "longtask" entry type.
 */
export class LongTaskCollector {
  private observer: PerformanceObserver | null = null;
  private totalCount = 0;
  private totalDurationMs = 0;
  private countAtLastSample = 0;
  private durationAtLastSample = 0;
  private recentTasks: LongTaskEntry[] = [];
  private windowMs: number;

  constructor(windowMs = 600_000) {
    this.windowMs = windowMs;
  }

  /** Check if long task observation is available. */
  static isAvailable(): boolean {
    if (typeof PerformanceObserver === "undefined") return false;
    try {
      const supported = PerformanceObserver.supportedEntryTypes;
      return supported?.includes("longtask") ?? false;
    } catch {
      return false;
    }
  }

  start(): void {
    if (!LongTaskCollector.isAvailable()) return;
    if (this.observer != null) return;

    this.totalCount = 0;
    this.totalDurationMs = 0;
    this.countAtLastSample = 0;
    this.durationAtLastSample = 0;
    this.recentTasks = [];

    this.observer = new PerformanceObserver((list) => {
      const now = performance.now();
      for (const entry of list.getEntries()) {
        this.totalCount++;
        this.totalDurationMs += entry.duration;
        this.recentTasks.push({ timestamp: now, duration: entry.duration });
      }
    });

    try {
      this.observer.observe({ entryTypes: ["longtask"] });
    } catch {
      // Some browsers may not support longtask observation
      this.observer = null;
    }
  }

  stop(): void {
    if (this.observer != null) {
      this.observer.disconnect();
      this.observer = null;
    }
  }

  reset(): void {
    this.totalCount = 0;
    this.totalDurationMs = 0;
    this.countAtLastSample = 0;
    this.durationAtLastSample = 0;
    this.recentTasks = [];
  }

  getCountSinceLastSample(): number {
    const count = this.totalCount - this.countAtLastSample;
    this.countAtLastSample = this.totalCount;
    return count;
  }

  getDurationSinceLastSample(): number {
    const duration = this.totalDurationMs - this.durationAtLastSample;
    this.durationAtLastSample = this.totalDurationMs;
    return duration;
  }

  getTotalCount(): number {
    return this.totalCount;
  }

  getTotalDurationMs(): number {
    return this.totalDurationMs;
  }

  getCountInWindow(): number {
    const now = performance.now();
    const cutoff = now - this.windowMs;
    // Remove entries older than window
    this.recentTasks = this.recentTasks.filter((t) => t.timestamp >= cutoff);
    return this.recentTasks.length;
  }

  getDurationInWindowMs(): number {
    const now = performance.now();
    const cutoff = now - this.windowMs;
    this.recentTasks = this.recentTasks.filter((t) => t.timestamp >= cutoff);
    return this.recentTasks.reduce((sum, t) => sum + t.duration, 0);
  }
}
