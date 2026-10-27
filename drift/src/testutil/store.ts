// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type EnhancedStore, type Reducer, type UnknownAction } from "@reduxjs/toolkit";

import { configureStore } from "@/configureStore";
import { type Action, reducer, type SliceState, type StoreState } from "@/state";
import { MockRuntime } from "@/testutil/runtime";
import { MAIN_WINDOW } from "@/window";

/** Configures a main-window store with no pre-render on the given runtime. */
export const configureMockStore = async (
  runtime: MockRuntime<StoreState> = new MockRuntime(true, { key: MAIN_WINDOW }),
): Promise<EnhancedStore<StoreState, UnknownAction | Action>> =>
  await configureStore<StoreState>({
    runtime,
    reducer: { drift: reducer as Reducer<SliceState> },
    enablePrerender: false,
  });
