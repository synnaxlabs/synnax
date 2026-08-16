// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project, type Synnax as Client } from "@synnaxlabs/client";
import { Panel, Status, Synnax } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/session";

const DEFAULT_NAME = "New Panel";

// A minted panel takes its tab's name. A resource whose name cannot be read still
// moves; only the name falls back.
const resolveName = async (client: Client | null, tab: panel.Tab): Promise<string> => {
  if (client == null || tab.variant !== "resource") return DEFAULT_NAME;
  try {
    return (await client.ontology.retrieve(tab.resource)).name;
  } catch {
    return DEFAULT_NAME;
  }
};

/** A tab together with the panel currently holding it. */
export interface TabOrigin extends Panel.TabDragPayload {}

export interface MintPanelForTab {
  (origin: TabOrigin): Promise<panel.Key | undefined>;
}

/**
 * useMintPanelForTab moves a tab into a panel created to hold it. The panel is an
 * ordinary project panel: it appears in the strip and outlives whatever shows it.
 * Nothing is selected, so callers decide where the new panel is shown.
 * @returns a callback resolving with the new panel's key, or undefined when the
 * create failed and the tab stayed where it was.
 */
export const useMintPanelForTab = (): MintPanelForTab => {
  const projectKey = Session.Project.useSelectSelected();
  const { updateAsync: create } = Panel.useCreate();
  const { dispatch } = Panel.useDispatch();
  const client = Synnax.use();
  return useCallback(
    async ({ panel: source, tab }) => {
      const key = uuid.create();
      const created = await create({
        key,
        name: await resolveName(client, tab),
        parent: project.ontologyID(projectKey),
        root: { variant: "leaf", tabs: [tab] },
      });
      if (!created) return undefined;
      dispatch({ key: source, actions: panel.removeTab({ key: tab.key }) });
      return key;
    },
    [create, dispatch, client, projectKey],
  );
};

export interface MoveTab {
  (origin: TabOrigin, destination: panel.Key): void;
}

/**
 * useMoveTab moves a tab into an existing panel and follows it there: the destination
 * panel is selected in this window, and the tab is selected within it.
 */
export const useMoveTab = (): MoveTab => {
  const move = Panel.useMoveTabToPanel();
  const dispatch = useDispatch();
  const selectTab = Session.Panel.useSelectTab();
  const handleError = Status.useErrorHandler();
  return useCallback(
    ({ panel: source, tab }, destination) =>
      handleError(async () => {
        if (source === destination) return;
        const landed = await move({ source, destination, tab });
        if (landed == null) return;
        dispatch(Session.Panel.select({ key: destination }));
        selectTab(landed, destination);
      }, "Failed to move the component"),
    [move, dispatch, selectTab, handleError],
  );
};

export interface MoveTabToNewPanel {
  (origin: TabOrigin): void;
}

/**
 * useMoveTabToNewPanel moves a tab into a panel created to hold it and selects that
 * panel in this window. The in-window counterpart of tearing a tab onto the desktop.
 */
export const useMoveTabToNewPanel = (): MoveTabToNewPanel => {
  const mint = useMintPanelForTab();
  const dispatch = useDispatch();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (origin) =>
      handleError(async () => {
        const key = await mint(origin);
        if (key != null) dispatch(Session.Panel.select({ key }));
      }, "Failed to move the component into a new panel"),
    [mint, dispatch, handleError],
  );
};
