// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { useCallback, useSyncExternalStore } from "react";

import { context } from "@/context";
import { type Store } from "@/schematic/edge/common/jumps/store";

const [Context, useStore] = context.create<Store>({
  displayName: "Jumps.Context",
  providerName: "Schematic.Diagram",
});

/// @brief provides the per-edge hop store to descendant edges. Render directly as
/// `<Context value={store}>`.
export { Context };

/// @brief returns the hop points for the edge with the given key, subscribing to only
/// that edge's hops. The returned array is reference-stable until the edge's hops change.
export const useCrossings = (key: string): xy.XY[] => {
  const store = useStore("useCrossings");
  const subscribe = useCallback(
    (onChange: () => void) => store.subscribe(key, onChange),
    [store, key],
  );
  const getSnapshot = useCallback(() => store.get(key), [store, key]);
  return useSyncExternalStore(subscribe, getSnapshot);
};
