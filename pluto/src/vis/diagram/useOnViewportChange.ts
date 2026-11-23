// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useReactFlowStoreApi, type Viewport, type ViewportStore } from "@xyflow/react";
import { useEffect } from "react";

/**
 * Calls `onChange` with every viewport React Flow renders, including the one it mounts
 * at, and with nothing while React Flow is unmounted. Reads the store transform rather
 * than `useOnViewportChange`, whose callbacks React Flow drops when it unmounts.
 */
export const useOnViewportChange = (onChange: (viewport: Viewport) => void): void => {
  const { optionsStore, viewportStore } = useReactFlowStoreApi();
  useEffect(() => {
    const sync = ({ transform: [x, y, zoom] }: ViewportStore): void => {
      // React Flow resets the viewport store before the options store when it
      // unmounts, so only a detached pane marks the placeholder transform.
      if (optionsStore.getState().domNode?.isConnected !== true) return;
      onChange({ x, y, zoom });
    };
    sync(viewportStore.getState());
    return viewportStore.subscribe((state, prev) => {
      if (state.transform !== prev.transform) sync(state);
    });
  }, [optionsStore, viewportStore, onChange]);
};
