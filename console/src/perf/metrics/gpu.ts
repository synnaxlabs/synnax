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

export class GpuCollector extends PollingCollector<number> {
  protected isAvailable(): boolean {
    return runtime.IS_TAURI;
  }

  protected async fetchValue(): Promise<number | null> {
    try {
      return await invoke<number | null>("get_gpu_usage");
    } catch {
      return null;
    }
  }

  static isAvailable(): boolean {
    return runtime.IS_TAURI;
  }

  getGpuPercent(): number | null {
    return this.getValue();
  }
}
