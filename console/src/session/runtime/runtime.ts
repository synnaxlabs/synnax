// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type UnknownAction } from "@reduxjs/toolkit";
import { NoopRuntime, type Runtime, type StoreState } from "@synnaxlabs/drift";
import { TauriRuntime } from "@synnaxlabs/drift/tauri";
import { isTauri } from "@tauri-apps/api/core";

export type Engine = "tauri" | "web";

export const ENGINE: Engine = isTauri() ? "tauri" : "web";

// The same as the main window's in tauri.conf.json.
const TRAFFIC_LIGHT_POSITION = { x: 12, y: 23 };

/** Creates the Drift runtime for the engine the Console runs in. */
export const createDrift = <
  S extends StoreState,
  A extends Action = UnknownAction,
>(): Runtime<S, A> =>
  ENGINE === "tauri"
    ? new TauriRuntime<S, A>({ trafficLightPosition: TRAFFIC_LIGHT_POSITION })
    : new NoopRuntime<S, A>();

/** True when the app opens no `synnax://` links, so a copied link opens nothing. */
export const LINKS_DISABLED = DESKTOP;

/** True when the app runs its own Core, so it has no login and no Core to pick. */
export const CORE_EMBEDDED = DESKTOP;
