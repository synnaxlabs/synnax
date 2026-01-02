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

export interface MacroContext {
  store: RootStore;
  dispatch: Dispatch;
  client: Synnax | null;
  placer: Layout.Placer;
  createdLayoutKeys: string[];
  availableChannelKeys: number[];
}

export interface MacroStep {
  name: string;
  execute: (context: MacroContext) => Promise<void>;
}

export interface MacroConfig {
  macros: MacroType[];
  iterations: number;
  delayBetweenMacrosMs: number;
  delayBetweenStepsMs: number;
}

export const DEFAULT_MACRO_CONFIG: MacroConfig = {
  macros: [BUILTIN_MACRO_TYPES.linePlot, BUILTIN_MACRO_TYPES.schematic],
  iterations: 10,
  delayBetweenMacrosMs: 50,
  delayBetweenStepsMs: 25,
};

export interface MacroResult {
  macroType: MacroType;
  startTime: number;
  endTime: number;
  durationMs: number;
  error?: string;
}
