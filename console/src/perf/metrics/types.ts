// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** A single metric sample collected at a point in time. */
export interface MetricSample {
  /** High-resolution timestamp from performance.now() */
  timestamp: number;
  /** CPU usage percentage (Tauri only, null in browser) */
  cpuPercent: number | null;
  /** JS heap used in MB (Chrome/Chromium only, null otherwise) */
  heapUsedMB: number | null;
  /** JS heap total in MB (Chrome/Chromium only, null otherwise) */
  heapTotalMB: number | null;
  /** Measured frames per second */
  frameRate: number;
  /** Number of long tasks (>50ms) since last sample */
  longTaskCount: number;
  /** Total duration of long tasks in ms since last sample */
  longTaskDurationMs: number;
  /** Number of network requests since last sample */
  networkRequestCount: number;
}

/** A heap snapshot for memory leak detection. */
export interface HeapSnapshot {
  /** Timestamp when snapshot was taken */
  timestamp: number;
  /** JS heap used in MB */
  heapUsedMB: number;
  /** JS heap total in MB */
  heapTotalMB: number;
}

/** Configuration for metrics collection. */
export interface MetricsConfig {
  /** How often to collect metric samples in milliseconds (default: 1000) */
  sampleIntervalMs: number;
  /** How often to capture heap snapshots in milliseconds (default: 60000) */
  heapSnapshotIntervalMs: number;
  /** Whether to track long tasks via PerformanceObserver (default: true) */
  enableLongTaskObserver: boolean;
  /** Whether to track network requests (default: true) */
  enableNetworkTracking: boolean;
}

/** Default metrics configuration. */
export const DEFAULT_METRICS_CONFIG: MetricsConfig = {
  sampleIntervalMs: 1000,
  heapSnapshotIntervalMs: 60000,
  enableLongTaskObserver: true,
  enableNetworkTracking: true,
};

/** Aggregated metrics snapshot containing all samples and heap snapshots. */
export interface MetricsSnapshot {
  /** All collected metric samples */
  samples: MetricSample[];
  /** Heap snapshots for leak detection */
  heapSnapshots: HeapSnapshot[];
  /** Start time of collection */
  startTime: number;
  /** End time of collection (null if still running) */
  endTime: number | null;
}

/** Creates an empty metrics snapshot. */
export const ZERO_METRICS_SNAPSHOT: MetricsSnapshot = {
  samples: [],
  heapSnapshots: [],
  startTime: 0,
  endTime: null,
};
