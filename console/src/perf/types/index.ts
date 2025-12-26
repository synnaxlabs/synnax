// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/perf/types/v0";

export type HarnessStatus = v0.HarnessStatus;
export type HarnessConfig = v0.HarnessConfig;
export type MetricsConfig = v0.MetricsConfig;
export type MacroConfig = v0.MacroConfig;
export type SliceState = v0.SliceState;
export type CpuReport = v0.CpuReport;
export type FpsReport = v0.FpsReport;
export type GpuReport = v0.GpuReport;
export type LeakReport = v0.LeakReport;

export const harnessStatusZ = v0.harnessStatusZ;
export const DEFAULT_HARNESS_CONFIG = v0.DEFAULT_HARNESS_CONFIG;
export const ZERO_SLICE_STATE = v0.ZERO_SLICE_STATE;
export const VERSION = v0.VERSION;

export type AnySliceState = v0.SliceState;

export const SLICE_MIGRATIONS: migrate.Migrations = {};

export const migrateSlice = migrate.migrator<AnySliceState, v0.SliceState>({
  name: "perf.slice",
  migrations: SLICE_MIGRATIONS,
  def: v0.ZERO_SLICE_STATE,
});
