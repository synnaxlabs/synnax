// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ontology,
  type panel,
  type project,
  type Synnax,
} from "@synnaxlabs/client";
import { createPanelParent } from "@synnaxlabs/client/testutil";
import { Panel as PPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";

import { Session } from "@/session";
import {
  createConsoleWrapper,
  getBySelector,
  type TestStore,
  uniqueName,
} from "@/testutil";

/** The rendered mosaic leaf: the drop target for both tabs and OS files. */
export const getMosaicLeaf = (): HTMLElement =>
  getBySelector<HTMLElement>(document, ".pluto-mosaic__leaf");

/** Persists a panel with the given tree to the Core, parented to a throwaway project.
 * */
export const createServerPanel = async (
  client: Synnax,
  root: panel.New["root"],
): Promise<panel.Panel> =>
  await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    root,
    parent: await createPanelParent(client),
  });

export interface ResourceTab {
  panelKey: panel.Key;
  tabKey: panel.TabKey;
}

/**
 * Creates a single-leaf panel holding one resource tab for the given resource, so the
 * panel scope hooks a mounted tab content reads (useTabResource) resolve to it.
 */
export const createResourceTab = async (
  client: Synnax,
  resource: ontology.ID,
): Promise<ResourceTab> => {
  const tabKey = uuid.create();
  const { key: panelKey } = await client.panels.create({
    name: uniqueName("panel"),
    root: { variant: "leaf", tabs: [{ variant: "resource", key: tabKey, resource }] },
    parent: await createPanelParent(client),
  });
  return { panelKey, tabKey };
};

export interface PanelWrapperParams {
  client: Synnax;
  store?: TestStore;
  panelKey?: panel.Key;
  tabKey?: panel.TabKey;
  /** Project selected in the session. Created on the Core when omitted. */
  project?: project.Key;
}

/**
 * Builds a console wrapper nested in the Panel (and optional Tab) scope, with a project
 * selected so panel creation can parent to it. The flux store is shared across every
 * mount of the returned wrapper, so callers prime a panel into the cache with
 * {@link primePanel} before rendering scope-reading components.
 */
export const createPanelWrapper = async ({
  client,
  store,
  panelKey,
  tabKey,
  project,
}: PanelWrapperParams): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const { wrapper: Console, store: resolvedStore } = await createConsoleWrapper({
    client,
    store,
  });
  const projectKey =
    project ??
    (await client.projects.create({ name: uniqueName("project"), layout: {} })).key;
  resolvedStore.dispatch(Session.Project.select(projectKey));
  // A scoped panel renders because it is the window's selected panel; tab
  // focus/visibility selectors require that term.
  if (panelKey != null) resolvedStore.dispatch(Session.Panel.select({ key: panelKey }));
  const wrapper = ({ children }: PropsWithChildren): ReactElement => {
    let inner: ReactNode = children;
    if (tabKey != null)
      inner = (
        <PPanel.TabScope.Provider value={tabKey}>{inner}</PPanel.TabScope.Provider>
      );
    if (panelKey != null)
      inner = <PPanel.Scope.Provider value={panelKey}>{inner}</PPanel.Scope.Provider>;
    return <Console>{inner}</Console>;
  };
  return { wrapper, store: resolvedStore };
};

/**
 * Retrieves the panel into the shared flux cache through the production query path, so
 * scope-reading selectors resolve it without a cache miss on their first render. The
 * cached document outlives the prime mount, so callers may unmount and render the unit
 * under test in a fresh mount of the same wrapper.
 */
export const primePanel = async (
  wrapper: FC<PropsWithChildren>,
  key: panel.Key,
): Promise<void> => {
  const { result, unmount } = renderHook(() => PPanel.useResult({ key }).data, {
    wrapper,
  });
  await waitFor(() => {
    if (result.current == null) throw new Error(`panel ${key} failed to load`);
  });
  unmount();
};
