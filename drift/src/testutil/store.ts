// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { EnhancedStore, Reducer, UnknownAction } from "@reduxjs/toolkit";

import { configureStore, type ConfigureStoreOptions } from "@/configureStore";
import { type Action, reducer, type SliceState, type StoreState } from "@/state";
import { MockRuntime } from "@/testutil/runtime";
import { MAIN_WINDOW } from "@/window";

// The checks still run. Only their timing warnings, which track machine load, go.
export const STORE_CHECKS = {
  immutableCheck: { warnAfter: Infinity },
  serializableCheck: { warnAfter: Infinity },
};

/** Configures a store with no pre-render on the given runtime, the main window's. */
export const configureMockStore = async (
  runtime: MockRuntime<StoreState> = new MockRuntime(true, { key: MAIN_WINDOW }),
  opts: Partial<ConfigureStoreOptions<StoreState>> = {},
): Promise<EnhancedStore<StoreState, UnknownAction | Action>> =>
  await configureStore<StoreState>({
    runtime,
    reducer: { drift: reducer as Reducer<SliceState> },
    enablePrerender: false,
    middleware: (getDefault) => getDefault(STORE_CHECKS),
    ...opts,
  });
