// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";

import { type Layout } from "@/layout";
import { type RootStore } from "@/store";

export type MacroType = string & { readonly __brand?: "MacroType" };

export const BUILTIN_MACRO_TYPES = {
  linePlot: "linePlot" as MacroType,
  schematic: "schematic" as MacroType,
} as const;

/** Context provided to macro steps for execution. */
export interface MacroContext {
  /** Redux store for state access */
  store: RootStore;
  /** Redux dispatch function */
  dispatch: Dispatch;
  /** Synnax client for data access */
  client: Synnax | null;
  /** Layout placer for creating visualizations */
  placer: Layout.Placer;
  /** Keys of layouts created during this harness run */
  createdLayoutKeys: string[];
  /** Channel keys available for use */
  availableChannelKeys: number[];
}

/** A single step in a macro. */
export interface MacroStep {
  /** Human-readable name of the step */
  name: string;
  /** Execute the step */
  execute: (context: MacroContext) => Promise<void>;
  /** Optional delay in ms after this step completes */
  delayAfterMs?: number;
}

/** Configuration for the macro runner. */
export interface MacroConfig {
  /** Macros to execute in each iteration */
  macros: MacroType[];
  /** Number of iterations (-1 for unlimited) */
  iterations: number;
  /** Delay between macro iterations in ms */
  delayBetweenIterationsMs: number;
  /** Delay between individual macros in ms */
  delayBetweenMacrosMs: number;
}

export const DEFAULT_MACRO_CONFIG: MacroConfig = {
  macros: [BUILTIN_MACRO_TYPES.linePlot, BUILTIN_MACRO_TYPES.schematic],
  iterations: 1,
  delayBetweenIterationsMs: 2000,
  delayBetweenMacrosMs: 1000,
};

/** Result of executing a single macro. */
export interface MacroResult {
  /** Type of macro executed */
  macroType: MacroType;
  /** Start timestamp (performance.now()) */
  startTime: number;
  /** End timestamp (performance.now()) */
  endTime: number;
  /** Duration in milliseconds */
  durationMs: number;
  /** Error message if the macro failed */
  error?: string;
}
