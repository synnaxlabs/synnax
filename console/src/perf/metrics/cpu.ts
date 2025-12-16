// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { invoke, isTauri } from "@tauri-apps/api/core";

/**
 * Collects CPU usage metrics via Tauri's sysinfo.
 * Only available in Tauri environment.
 */
export class CpuCollector {
  private cachedCpuPercent: number | null = null;
  private useTauri: boolean;
  private updateInterval: ReturnType<typeof setInterval> | null = null;

  constructor() {
    this.useTauri = isTauri();
  }

  /** Start the collector (fetches Tauri CPU usage periodically). */
  start(): void {
    if (!this.useTauri) return;

    // Fetch immediately
    void this.fetchTauriCpu();
    // Then poll every second
    this.updateInterval = setInterval(() => {
      void this.fetchTauriCpu();
    }, 1000);
  }

  /** Stop the collector. */
  stop(): void {
    if (this.updateInterval != null) {
      clearInterval(this.updateInterval);
      this.updateInterval = null;
    }
  }

  /** Fetch CPU usage from Tauri backend. */
  private async fetchTauriCpu(): Promise<void> {
    try {
      const percent = await invoke<number>("get_cpu_usage");
      this.cachedCpuPercent = percent;
    } catch {
      this.cachedCpuPercent = null;
    }
  }

  /** Check if CPU metrics are available. */
  static isAvailable(): boolean {
    return isTauri();
  }

  /** Get current CPU usage as a percentage, or null if not available. */
  getCpuPercent(): number | null {
    if (!this.useTauri) return null;
    return this.cachedCpuPercent;
  }
}
