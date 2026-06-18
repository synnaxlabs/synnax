// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { createContext, useCallback, useContext, useSyncExternalStore } from "react";

import { type Store } from "@/schematic/edge/common/jumps/store";

const Context = createContext<Store | null>(null);
Context.displayName = "Jumps.Context";

/// @brief provides the per-edge hop store to descendant edges. When absent, edges
/// render with no hops, so it doubles as the feature's on/off switch.
export const Provider = Context.Provider;

const EMPTY: xy.XY[] = [];

/// @brief returns the hop points for the edge with the given key, subscribing to only
/// that edge's hops. Returns an empty list when no Provider is mounted. The returned
/// array is reference-stable until the edge's hops change.
export const useCrossings = (key: string): xy.XY[] => {
  const store = useContext(Context);
  const subscribe = useCallback(
    (onChange: () => void) => store?.subscribe(key, onChange) ?? (() => {}),
    [store, key],
  );
  const getSnapshot = useCallback(() => store?.get(key) ?? EMPTY, [store, key]);
  return useSyncExternalStore(subscribe, getSnapshot);
};
