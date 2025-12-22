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
 * Collects GPU usage metrics via Tauri's NVML integration.
 * Only available in Tauri environment on Windows/Linux with NVIDIA GPUs.
 * Returns null on macOS or when no compatible GPU is available.
 */
export class GpuCollector {
  private cachedGpuPercent: number | null = null;
  private updateInterval: ReturnType<typeof setInterval> | null = null;

  start(): void {
    if (!runtime.IS_TAURI) return;

    // Fetch immediately
    void this.fetchTauriGpu();
    // Then poll every second
    this.updateInterval = setInterval(() => {
      void this.fetchTauriGpu();
    }, 1000);
  }

  stop(): void {
    if (this.updateInterval != null) {
      clearInterval(this.updateInterval);
      this.updateInterval = null;
    }
  }

  private async fetchTauriGpu(): Promise<void> {
    try {
      const percent = await invoke<number | null>("get_gpu_usage");
      this.cachedGpuPercent = percent;
    } catch {
      this.cachedGpuPercent = null;
    }
  }

  static isAvailable(): boolean {
    return runtime.IS_TAURI;
  }

  getGpuPercent(): number | null {
    if (!runtime.IS_TAURI) return null;
    return this.cachedGpuPercent;
  }

  reset(): void {
    this.cachedGpuPercent = null;
  }
}
