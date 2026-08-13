// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project, type Synnax as Client } from "@synnaxlabs/client";
import { type Haul, Mosaic, Panel, Status, Synnax } from "@synnaxlabs/pluto";
import { uuid, xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { isPillHaulItem } from "@/feature/panel/haul";
import { useOpenWindow } from "@/feature/panel/useOpenWindow";
import { Window } from "@/platform/window";
import { Session } from "@/session";

// Places the new window under the cursor rather than centered on it, so the torn tab
// lands where the pointer released it.
const OFFSET: xy.XY = { x: -80, y: -45 };

const DEFAULT_NAME = "New Panel";

// The panel takes the torn tab's name. A resource whose name cannot be read still tears
// off; only the name falls back.
const resolveName = async (client: Client | null, tab: panel.Tab): Promise<string> => {
  if (client == null || tab.variant !== "resource") return DEFAULT_NAME;
  try {
    return (await client.ontology.retrieve(tab.resource)).name;
  } catch {
    return DEFAULT_NAME;
  }
};

export interface TearOffTab {
  (tab: Panel.TabDragPayload, position?: xy.XY): void;
}

/**
 * useTearOffTab moves a tab into a panel minted to hold it and opens a window showing
 * that panel. The panel is an ordinary project panel: it appears in the strip and
 * outlives the window.
 */
export const useTearOffTab = (): TearOffTab => {
  const openWindow = useOpenWindow();
  const projectKey = Session.Project.useSelectSelected();
  const { updateAsync: create } = Panel.useCreate();
  const { dispatch } = Panel.useDispatch();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return useCallback(
    ({ panel: sourceKey, tab }, position) =>
      handleError(async () => {
        const key = uuid.create();
        const created = await create({
          key,
          name: await resolveName(client, tab),
          parent: project.ontologyID(projectKey),
          root: { variant: "leaf", tabs: [tab] },
        });
        if (!created) return;
        dispatch({ key: sourceKey, actions: panel.removeTab({ key: tab.key }) });
        openWindow(key, { position });
      }, "Failed to open the component in a new window"),
    [openWindow, create, dispatch, client, projectKey, handleError],
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
