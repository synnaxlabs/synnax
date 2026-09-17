// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactFlowState, useStoreApi, type Viewport } from "@xyflow/react";
import { useEffect } from "react";

/**
 * Calls `onChange` with every viewport React Flow renders, including the one it mounts
 * at. Subscribes to the store transform that positions the DOM instead of using React
 * Flow's `useOnViewportChange`: that hook registers its callbacks in the store once,
 * and React Flow resets the store when it unmounts, so a remount reports nothing.
 * Nothing is reported while React Flow is unmounted, when the transform is a
 * placeholder.
 */
export const useOnViewportChange = (onChange: (viewport: Viewport) => void): void => {
  const store = useStoreApi();
  useEffect(() => {
    const sync = ({ panZoom, transform: [x, y, zoom] }: ReactFlowState): void => {
      if (panZoom == null) return;
      onChange({ x, y, zoom });
    };
    sync(store.getState());
    return store.subscribe((state, prev) => {
      if (state.transform !== prev.transform) sync(state);
    });
  }, [store, onChange]);
};
