// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Haul, Mosaic, Panel, Status } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { isPillHaulItem } from "@/feature/panel/haul";
import { type TabOrigin, useMintPanelForTab } from "@/feature/panel/useMoveTab";
import { useOpenWindow } from "@/feature/panel/useOpenWindow";
import { Window } from "@/platform/window";

// Places the new window under the cursor rather than centered on it, so the torn tab
// lands where the pointer released it.
const OFFSET: xy.XY = { x: -80, y: -45 };

export interface TearOffTab {
  (origin: TabOrigin, position?: xy.XY): void;
}

/**
 * useTearOffTab moves a tab into a panel minted to hold it and opens a window showing
 * that panel. The source window is left on the panel it was already showing.
 */
export const useTearOffTab = (): TearOffTab => {
  const openWindow = useOpenWindow();
  const mint = useMintPanelForTab();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (origin, position) =>
      handleError(async () => {
        const key = await mint(origin);
        if (key != null) openWindow(key, { position });
      }, "Failed to open the component in a new window"),
    [openWindow, mint, handleError],
  );
};

const canDrop: Haul.CanDrop = ({ items }) =>
  items.length === 1 &&
  (Mosaic.isTabDropHaulItem(items[0]) || isPillHaulItem(items[0]));

/**
 * useTearOff opens a window for content released over the desktop: a panel pill opens a
 * second window on that panel, a tab tears off into a panel of its own. Mount once per
 * window.
 */
export const useTearOff = (): void => {
  const openWindow = useOpenWindow();
  const tearOffTab = useTearOffTab();
  const handleDrop = useCallback<Window.OnDropOutside>(
    ({ items: [item] }, cursor) => {
      const position = xy.translate(cursor, OFFSET);
      if (isPillHaulItem(item)) return openWindow(item.key, { position });
      const dragged = Panel.parseTabDragPayload(item.data);
      if (dragged != null) tearOffTab(dragged, position);
    },
    [openWindow, tearOffTab],
  );
  Window.useDropOutside({ canDrop, onDrop: handleDrop });
};
