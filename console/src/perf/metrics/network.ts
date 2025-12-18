// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MetricTableColumn } from "@/perf/components/MetricTable";
import { MAX_STORED_ENDPOINTS, TEXT_ROW_COLOR } from "@/perf/constants";
import { formatDuration, truncateEndpoint } from "@/perf/utils/formatting";

export interface EndpointStats {
  endpoint: string;
  count: number;
  avgDurationMs: number;
  totalDurationMs: number;
  lastSeenMs: number;
}

export const NETWORK_TABLE_COLUMNS: MetricTableColumn<EndpointStats>[] = [
  { getValue: (ep) => truncateEndpoint(ep.endpoint), color: TEXT_ROW_COLOR },
  { getValue: (ep) => formatDuration(ep.avgDurationMs), color: TEXT_ROW_COLOR },
  { getValue: (ep) => ep.count, color: TEXT_ROW_COLOR },
];

export const getNetworkTableKey = (ep: EndpointStats): string => ep.endpoint;

export const getNetworkTableTooltip = (ep: EndpointStats): string => ep.endpoint;

/**
 * Normalizes a URL to an endpoint path for aggregation.
 * Strips query params, fragments, and protocol/host.
 */
const normalizeToEndpoint = (url: string): string => {
  try {
    const parsed = new URL(url);
    return parsed.pathname;
  } catch {
    const pathMatch = url.match(/^(?:https?:\/\/[^/]+)?([^?#]*)/);
    return pathMatch?.[1] ?? url;
  }
};

/**
 * Tracks network requests using PerformanceObserver.
 * Supports both simple counting and detailed endpoint profiling.
 * Supported in all Tauri 2.x webviews (WebView2 on Windows, Safari 15+ WebKit on macOS/Linux).
 */
export class NetworkCollector {
  private totalCount = 0;
  private countAtLastSample = 0;
  private observer: PerformanceObserver | null = null;

  // Endpoint profiling data
  private endpointCounts = new Map<string, number>();
  private endpointDurations = new Map<string, number>();
  private endpointLastSeen = new Map<string, number>();

  start(): void {
    if (this.observer != null) return;
    this.totalCount = 0;
    this.countAtLastSample = 0;

    this.observer = new PerformanceObserver((list) => {
      const entries = list.getEntries() as PerformanceResourceTiming[];
      this.totalCount += entries.length;
      const now = performance.now();

      for (const entry of entries) {
        const endpoint = normalizeToEndpoint(entry.name);
        this.endpointCounts.set(
          endpoint,
          (this.endpointCounts.get(endpoint) ?? 0) + 1,
        );
        this.endpointDurations.set(
          endpoint,
          (this.endpointDurations.get(endpoint) ?? 0) + entry.duration,
        );
        this.endpointLastSeen.set(endpoint, now);
      }
    });
    this.observer.observe({ entryTypes: ["resource"] });
  }

  stop(): void {
    if (this.observer == null) return;
    this.observer.disconnect();
    this.observer = null;
  }

  reset(): void {
    this.totalCount = 0;
    this.countAtLastSample = 0;
    this.endpointCounts.clear();
    this.endpointDurations.clear();
    this.endpointLastSeen.clear();
  }

  getCountSinceLastSample(): number {
    const count = this.totalCount - this.countAtLastSample;
    this.countAtLastSample = this.totalCount;
    return count;
  }

  getTotalCount(): number {
    return this.totalCount;
  }

  /**
   * Get endpoint statistics sorted by request count (descending).
   * Also performs automatic cleanup to prevent memory leaks when too many unique endpoints are stored.
   * @returns Object with data array and total count
   */
  getTopEndpoints(): { data: EndpointStats[]; total: number; truncated: boolean } {
    // Clean up low-count endpoints if we have too many stored
    if (this.endpointCounts.size > MAX_STORED_ENDPOINTS) 
      this.cleanupEndpoints(MAX_STORED_ENDPOINTS);
    

    const stats: EndpointStats[] = [];

    for (const [endpoint, count] of this.endpointCounts) {
      const totalDurationMs = this.endpointDurations.get(endpoint) ?? 0;
      const lastSeenMs = this.endpointLastSeen.get(endpoint) ?? 0;
      stats.push({
        endpoint,
        count,
        totalDurationMs,
        avgDurationMs: count > 0 ? totalDurationMs / count : 0,
        lastSeenMs,
      });
    }

    // Sort by count (descending), then by last seen (descending)
    stats.sort((a, b) => {
      if (a.count !== b.count) return b.count - a.count;
      return b.lastSeenMs - a.lastSeenMs;
    });

    return {
      data: stats,
      total: stats.length,
      truncated: false,
    };
  }

  /**
   * Removes low-count endpoints to prevent unbounded memory growth.
   * @param maxToKeep Maximum number of endpoints to keep
   */
  private cleanupEndpoints(maxToKeep: number): void {
    // Build array of endpoints sorted by count (descending)
    const sorted = Array.from(this.endpointCounts.entries())
      .sort((a, b) => b[1] - a[1]);

    // Keep only the top N endpoints
    const toKeep = new Set(
      sorted.slice(0, maxToKeep).map(([endpoint]) => endpoint)
    );

    // Remove endpoints not in the top set
    for (const endpoint of this.endpointCounts.keys()) 
      if (!toKeep.has(endpoint)) {
        this.endpointCounts.delete(endpoint);
        this.endpointDurations.delete(endpoint);
        this.endpointLastSeen.delete(endpoint);
      }
    
  }

  getEndpointCount(): number {
    return this.endpointCounts.size;
  }
}
