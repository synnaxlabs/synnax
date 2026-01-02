// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const DEFAULT_POLL_INTERVAL_MS = 1000;

export abstract class PollingCollector<T> {
  protected cachedValue: T | null = null;
  private updateInterval: ReturnType<typeof setInterval> | null = null;

  constructor(private readonly pollIntervalMs: number = DEFAULT_POLL_INTERVAL_MS) {}

  protected abstract fetchValue(): Promise<T | null>;
  protected abstract isAvailable(): boolean;

  start(): void {
    if (!this.isAvailable()) return;
    void this.fetch();
    this.updateInterval = setInterval(() => void this.fetch(), this.pollIntervalMs);
  }

  stop(): void {
    if (this.updateInterval != null) {
      clearInterval(this.updateInterval);
      this.updateInterval = null;
    }
  }

  reset(): void {
    this.cachedValue = null;
  }

  protected getValue(): T | null {
    if (!this.isAvailable()) return null;
    return this.cachedValue;
  }

  private async fetch(): Promise<void> {
    this.cachedValue = await this.fetchValue();
  }
}
