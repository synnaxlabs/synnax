// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { panel, Synnax } from "@synnaxlabs/client";
import { createPanelProject } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, type FC, type PropsWithChildren, Suspense } from "react";

import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { getBySelector, getIconButton, type TestStore } from "@/testutil";

export interface TabWrapperParams {
  /** Creates the panel on the Core. */
  client: Synnax;
  /** The user the Console signs in as. Defaults to the client. */
  as?: Synnax;
  /** What the tab shows. A View.Frame reads its open view from it. */
  tab: panel.NewTab;
  /** Other tabs in the same leaf, after the tab. */
  beside?: panel.NewTab[];
  store?: TestStore;
}

export interface TabWrapper {
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
  panelKey: panel.Key;
  tabKey: panel.TabKey;
}

/**
 * Creates a panel on the Core with one tab, and a wrapper that renders its children in
 * that tab, with the panel already in the cache.
 */
export const createTabWrapper = async ({
  client,
  as = client,
  tab,
  beside = [],
  store,
}: TabWrapperParams): Promise<TabWrapper> => {
  const tabKey = uuid.create();
  const { key: panelKey } = await createServerPanel(client, {
    variant: "leaf",
    tabs: [{ ...tab, key: tabKey }, ...beside],
  });
  const { wrapper, store: resolved } = await createPanelWrapper({
    client: as,
    store,
    project: await createPanelProject(client),
    panelKey,
    tabKey,
  });
  await primePanel(wrapper, panelKey);
  // A tab suspends inside a boundary in the mosaic.
  const Tab: FC<PropsWithChildren> = ({ children }) =>
    createElement(wrapper, null, createElement(Suspense, null, children));
  return { wrapper: Tab, store: resolved, panelKey, tabKey };
};

/**
 * Waits for the icon button with the given icon name in the mounted View.Toolbar, and
 * returns it.
 */
export const findToolbarIconButton = async (icon: string): Promise<HTMLButtonElement> =>
  await waitFor(() =>
    getIconButton(getBySelector(document.body, ".console-view__toolbar"), icon),
  );

/**
 * Opens the filter menu of the mounted editable view, then the submenu of the property
 * with the name, and returns the picker it opens.
 */
export const openFilterProperty = async (name: string): Promise<HTMLElement> => {
  fireEvent.click(await waitFor(() => getIconButton(document.body, "filter")));
  fireEvent.click(await screen.findByRole("menuitem", { name }));
  return await screen.findByRole("dialog", { name });
};
