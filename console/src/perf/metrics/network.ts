// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Tracks network requests using PerformanceObserver.
 * Supported in all Tauri 2.x webviews (WebView2 on Windows, Safari 15+ WebKit on macOS/Linux).
 */
export class NetworkCollector {
  private totalCount = 0;
  private countAtLastSample = 0;
  private observer: PerformanceObserver | null = null;

  /** Start tracking network requests. */
  start(): void {
    if (this.observer != null) return;
    this.totalCount = 0;
    this.countAtLastSample = 0;

    this.observer = new PerformanceObserver((list) => {
      this.totalCount += list.getEntries().length;
    });
    this.observer.observe({ entryTypes: ["resource"] });
  }

  /** Stop tracking network requests. */
  stop(): void {
    if (this.observer == null) return;
    this.observer.disconnect();
    this.observer = null;
  }

  /** Get number of network requests since last sample call. */
  getCountSinceLastSample(): number {
    const count = this.totalCount - this.countAtLastSample;
    this.countAtLastSample = this.totalCount;
    return count;
  }

  /** Get total count of all network requests. */
  getTotalCount(): number {
    return this.totalCount;
  }
}
