// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { runtime } from "@synnaxlabs/x";
import { invoke } from "@tauri-apps/api/core";

import { PollingCollector } from "@/perf/metrics/polling-collector";
import { type HeapSnapshot } from "@/perf/metrics/types";

interface PerformanceMemory {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
}

interface PerformanceWithMemory extends Performance {
  memory?: PerformanceMemory;
}

const BYTES_TO_MB = 1024 * 1024;

export class HeapCollector extends PollingCollector<number> {
  protected isAvailable(): boolean {
    return runtime.IS_TAURI;
  }

  protected async fetchValue(): Promise<number | null> {
    try {
      const bytes = await invoke<number>("get_memory_usage");
      return bytes / BYTES_TO_MB;
    } catch {
      return null;
    }
  }

  static isAvailable(): boolean {
    return runtime.IS_TAURI || "memory" in performance;
  }

  getHeapUsedMB(): number | null {
    if (runtime.IS_TAURI) return this.getValue();
    const perf = performance as PerformanceWithMemory;
    if (perf.memory == null) return null;
    return perf.memory.usedJSHeapSize / BYTES_TO_MB;
  }

  getHeapTotalMB(): number | null {
    if (runtime.IS_TAURI) return this.getValue();
    const perf = performance as PerformanceWithMemory;
    if (perf.memory == null) return null;
    return perf.memory.totalJSHeapSize / BYTES_TO_MB;
  }

  getHeapLimitMB(): number | null {
    if (runtime.IS_TAURI) return null;
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
}
