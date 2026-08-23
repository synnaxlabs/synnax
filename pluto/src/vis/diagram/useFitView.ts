// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type xy } from "@synnaxlabs/x";
import { getViewportForBounds, useReactFlow, useStoreApi } from "@xyflow/react";
import { useCallback } from "react";

import { type diagram } from "@/vis/diagram/aether";
import { selectNodesScreenBounds } from "@/vis/diagram/util";

/**
 * Returns a replacement for React Flow's fitView that fits the viewport around
 * everything the nodes render, not just their measured boxes, so content placed
 * outside a node (a label) stays in view. Options follow React Flow's fitView.
 */
export const useFitView = (): ((options?: diagram.FitViewOptions) => void) => {
  const store = useStoreApi();
  const { screenToFlowPosition, setViewport } = useReactFlow();
  return useCallback(
    (options?: diagram.FitViewOptions) => {
      const { domNode, width, height, minZoom, maxZoom } = store.getState();
      if (domNode == null) return;
      const screen = selectNodesScreenBounds(domNode);
      if (screen == null) return;
      const toFlow = (p: xy.XY) => screenToFlowPosition(p, { snapToGrid: false });
      const min = toFlow(box.topLeft(screen));
      const max = toFlow(box.bottomRight(screen));
      const bounds = {
        x: min.x,
        y: min.y,
        width: max.x - min.x,
        height: max.y - min.y,
      };
      const viewport = getViewportForBounds(
        bounds,
        width,
        height,
        options?.minZoom ?? minZoom,
        options?.maxZoom ?? maxZoom,
        options?.padding ?? 0.1,
      );
      void setViewport(viewport, options);
    },
    [store, screenToFlowPosition, setViewport],
  );
};
