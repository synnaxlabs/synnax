// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MetricTableColumn } from "@/perf/components/MetricTable";

interface LongTaskEntry {
  timestamp: number;
  duration: number;
  name: string;
}

interface UserEvent {
  type: string;
  timestamp: number;
  target?: string;
}


export interface LongTaskStats {
  name: string;
  duration: number;
  timestamp: number;
  age: number;
}

const formatAge = (ageMs: number): string => {
  if (ageMs < 1000) return `${Math.floor(ageMs) }ms ago`;
  if (ageMs < 60000) return `${Math.floor(ageMs / 1000)}s ago`;
  return `${Math.floor(ageMs / 60000)} m ago`;
};

export const LONG_TASK_TABLE_COLUMNS: MetricTableColumn<LongTaskStats>[] = [
  { getValue: (task, _) => task.name, color: 7 },
  { getValue: (task, _) => formatAge(task.age) },
  { getValue: (task, _) => `${task.duration.toFixed(0)} ms` },
];

export const getLongTaskTableKey = (task: LongTaskStats, index: number): string =>
  `${task.timestamp}-${index}`;

/**
 * Tracks long tasks (>50ms) that block the main thread.
 * Uses the PerformanceObserver API with "longtask" entry type when available.
 * Falls back to requestAnimationFrame timing on platforms without native support (WebKit).
 */
export class LongTaskCollector {
  private observer: PerformanceObserver | null = null;
  private totalCount = 0;
  private totalDurationMs = 0;
  private countAtLastSample = 0;
  private durationAtLastSample = 0;
  private recentTasks: LongTaskEntry[] = [];
  private windowMs: number;

  // RAF fallback tracking
  private rafId: number | null = null;
  private lastRafTime: number | null = null;
  private useRafFallback = false;
  private readonly LONG_TASK_THRESHOLD_MS = 50;

  // Event tracking for task attribution
  private recentEvents: UserEvent[] = [];
  private readonly EVENT_CORRELATION_WINDOW_MS = 1000;
  private readonly MAX_TRACKED_EVENTS = 50;
  private eventListeners: Array<{ type: string; handler: EventListener }> = [];

  constructor(windowMs = 600_000) {
    this.windowMs = windowMs;
  }

  /** Check if native long task observation is available. */
  static isAvailable(): boolean {
    if (typeof PerformanceObserver === "undefined") return false;
    try {
      const supported = PerformanceObserver.supportedEntryTypes;
      return supported?.includes("longtask") ?? false;
    } catch {
      return false;
    }
  }

  private trackEvent(type: string, event: Event): void {
    const target = event.target instanceof Element ? event.target.tagName.toLowerCase() : undefined;

    this.recentEvents.push({
      type,
      timestamp: performance.now(),
      target,
    });

    if (this.recentEvents.length > this.MAX_TRACKED_EVENTS) {
      this.recentEvents.shift();
    }
  }

  private findEventForTask(taskTimestamp: number): string {
    const correlationStart = taskTimestamp - this.EVENT_CORRELATION_WINDOW_MS;

    for (let i = this.recentEvents.length - 1; i >= 0; i--) {
      const event = this.recentEvents[i];
      if (event.timestamp <= taskTimestamp && event.timestamp >= correlationStart) {
        return event.target ? `${event.type} (${event.target})` : event.type;
      }
    }

    return "Unknown";
  }

  private setupEventTracking(): void {
    if (typeof window === "undefined") return;

    const eventTypes = [
      "click",
      "keydown",
      "input",
      "submit",
      "dragstart",
      "dragend",
      "focus",
      "blur",
      "change",
      "paste",
      "scroll",
      "resize",
      "wheel",
      "touchstart",
      "touchend",
      "popstate",
      "contextmenu",
    ];

    for (const type of eventTypes) {
      const handler = (e: Event) => this.trackEvent(type, e);
      window.addEventListener(type, handler, { capture: true, passive: true });
      this.eventListeners.push({ type, handler });
    }
  }

  private cleanupEventTracking(): void {
    if (typeof window === "undefined") return;

    for (const { type, handler } of this.eventListeners) {
      window.removeEventListener(type, handler, { capture: true });
    }
    this.eventListeners = [];
    this.recentEvents = [];
  }

  start(): void {
    if (this.observer != null || this.rafId != null) return;

    this.totalCount = 0;
    this.totalDurationMs = 0;
    this.countAtLastSample = 0;
    this.durationAtLastSample = 0;
    this.recentTasks = [];
    this.lastRafTime = null;

    this.setupEventTracking();

    // Try native Long Task API first
    if (LongTaskCollector.isAvailable()) {
      this.useRafFallback = false;
      this.observer = new PerformanceObserver((list) => {
        const now = performance.now();
        for (const entry of list.getEntries()) {
          this.totalCount++;
          this.totalDurationMs += entry.duration;
          const name = this.findEventForTask(now);
          this.recentTasks.push({ timestamp: now, duration: entry.duration, name });
        }
      });

      try {
        this.observer.observe({ entryTypes: ["longtask"] });
        return;
      } catch {
        this.observer = null;
      }
    }

    // Fallback: Use requestAnimationFrame timing to detect blocking
    this.useRafFallback = true;
    this.startRafTracking();
  }

  private startRafTracking(): void {
    const rafCallback = (currentTime: number) => {
      if (this.lastRafTime !== null) {
        const delta = currentTime - this.lastRafTime;

        if (delta > this.LONG_TASK_THRESHOLD_MS) {
          const taskDuration = delta;
          const now = performance.now();
          this.totalCount++;
          this.totalDurationMs += taskDuration;
          const name = this.findEventForTask(now);
          this.recentTasks.push({
            timestamp: now,
            duration: taskDuration,
            name,
          });
        }
      }

      this.lastRafTime = currentTime;
      this.rafId = requestAnimationFrame(rafCallback);
    };

    this.rafId = requestAnimationFrame(rafCallback);
  }

  stop(): void {
    if (this.observer != null) {
      this.observer.disconnect();
      this.observer = null;
    }

    if (this.rafId != null) {
      cancelAnimationFrame(this.rafId);
      this.rafId = null;
      this.lastRafTime = null;
    }

    this.cleanupEventTracking();
  }

  reset(): void {
    this.totalCount = 0;
    this.totalDurationMs = 0;
    this.countAtLastSample = 0;
    this.durationAtLastSample = 0;
    this.recentTasks = [];
    this.lastRafTime = null;
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

  /**
   * Get the most recent long tasks sorted by timestamp (descending).
   * Also performs automatic cleanup of tasks outside the tracking window to prevent memory leaks.
   * @returns Object with data array and total count
   */
  getTopLongTasks(): { data: LongTaskStats[]; total: number; truncated: boolean } {
    const now = performance.now();
    const cutoff = now - this.windowMs;

    // Clean up old tasks to prevent memory leaks
    this.recentTasks = this.recentTasks.filter((t) => t.timestamp >= cutoff);

    const data = this.recentTasks
      .slice()
      .reverse()
      .map((task) => ({
        name: task.name,
        duration: task.duration,
        timestamp: task.timestamp,
        age: now - task.timestamp,
      }));

    return {
      data,
      total: data.length,
      truncated: false,
    };
  }
}
