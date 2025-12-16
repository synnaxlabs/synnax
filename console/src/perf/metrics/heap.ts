// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { invoke, isTauri } from "@tauri-apps/api/core";

import { type HeapSnapshot } from "@/perf/metrics/types";

// Chrome's non-standard memory API
interface PerformanceMemory {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
}

interface PerformanceWithMemory extends Performance {
  memory?: PerformanceMemory;
}

const BYTES_TO_MB = 1024 * 1024;

/**
 * Collects memory metrics.
 * In Tauri: Uses native process memory via Rust sysinfo.
 * In browser: Falls back to Chrome's performance.memory API.
 */
export class HeapCollector {
  private cachedMemoryMB: number | null = null;
  private useTauri: boolean;
  private updateInterval: ReturnType<typeof setInterval> | null = null;

  constructor() {
    this.useTauri = isTauri();
  }

  start(): void {
    if (this.useTauri) {
      // Fetch immediately
      void this.fetchTauriMemory();
      // Then poll every second
      this.updateInterval = setInterval(() => {
        void this.fetchTauriMemory();
      }, 1000);
    }
  }

  stop(): void {
    if (this.updateInterval != null) {
      clearInterval(this.updateInterval);
      this.updateInterval = null;
    }
  }

  private async fetchTauriMemory(): Promise<void> {
    try {
      const bytes = await invoke<number>("get_memory_usage");
      this.cachedMemoryMB = bytes / BYTES_TO_MB;
    } catch {
      this.cachedMemoryMB = null;
    }
  }

  static isAvailable(): boolean {
    return isTauri() || "memory" in performance;
  }

  getHeapUsedMB(): number | null {
    if (this.useTauri) return this.cachedMemoryMB;

    const perf = performance as PerformanceWithMemory;
    if (perf.memory == null) return null;
    return perf.memory.usedJSHeapSize / BYTES_TO_MB;
  }

  getHeapTotalMB(): number | null {
    if (this.useTauri)
      // Tauri returns process memory, not heap total - return same value
      return this.cachedMemoryMB;

    const perf = performance as PerformanceWithMemory;
    if (perf.memory == null) return null;
    return perf.memory.totalJSHeapSize / BYTES_TO_MB;
  }

  getHeapLimitMB(): number | null {
    if (this.useTauri) return null;

    const perf = performance as PerformanceWithMemory;
    if (perf.memory == null) return null;
    return perf.memory.jsHeapSizeLimit / BYTES_TO_MB;
  }

  captureSnapshot(): HeapSnapshot | null {
    const heapUsed = this.getHeapUsedMB();
    const heapTotal = this.getHeapTotalMB();
    if (heapUsed == null || heapTotal == null) return null;
    return {
      timestamp: performance.now(),
      heapUsedMB: heapUsed,
      heapTotalMB: heapTotal,
    };
  }

  reset(): void {
    this.cachedMemoryMB = null;
  }
}
