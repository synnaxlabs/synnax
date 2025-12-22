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

/**
 * Collects CPU usage metrics via Tauri's sysinfo.
 * Only available in Tauri environment.
 */
export class CpuCollector {
  private cachedCpuPercent: number | null = null;
  private updateInterval: ReturnType<typeof setInterval> | null = null;

  start(): void {
    if (!runtime.IS_TAURI) return;

    // Fetch immediately
    void this.fetchTauriCpu();
    // Then poll every second
    this.updateInterval = setInterval(() => {
      void this.fetchTauriCpu();
    }, 1000);
  }

  stop(): void {
    if (this.updateInterval != null) {
      clearInterval(this.updateInterval);
      this.updateInterval = null;
    }
  }

  private async fetchTauriCpu(): Promise<void> {
    try {
      const percent = await invoke<number>("get_cpu_usage");
      this.cachedCpuPercent = percent;
    } catch {
      this.cachedCpuPercent = null;
    }
  }

  static isAvailable(): boolean {
    return runtime.IS_TAURI;
  }

  getCpuPercent(): number | null {
    if (!runtime.IS_TAURI) return null;
    return this.cachedCpuPercent;
  }

  reset(): void {
    this.cachedCpuPercent = null;
  }
}
