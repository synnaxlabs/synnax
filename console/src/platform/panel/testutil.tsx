// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, type Synnax } from "@synnaxlabs/client";
import { Panel as PPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";

import { createConsoleWrapper, type TestStore, uniqueName } from "@/testutil";

/** Persists a panel with the given tree to the cluster. */
export const createServerPanel = async (
  client: Synnax,
  root: panel.New["root"],
): Promise<panel.Panel> =>
  await client.panels.create({ key: uuid.create(), name: uniqueName("panel"), root });

export interface PanelWrapperParams {
  client: Synnax;
  store?: TestStore;
  panelKey?: panel.Key;
  tabKey?: panel.TabKey;
}

/**
 * Builds a console wrapper nested in the Panel (and optional Tab) scope. The flux store
 * is shared across every mount of the returned wrapper, so callers prime a panel into
 * the cache with {@link primePanel} before rendering scope-reading components.
 */
export const createPanelWrapper = async ({
  client,
  store,
  panelKey,
  tabKey,
}: PanelWrapperParams): Promise<{ wrapper: FC<PropsWithChildren>; store: TestStore }> => {
  const { wrapper: Console, store: resolvedStore } = await createConsoleWrapper({
    client,
    store,
  });
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
  const { result, unmount } = renderHook(() => PPanel.useRetrieve({ key }), {
    wrapper,
  });
  await waitFor(() => {
    if (result.current.variant !== "success")
      throw new Error(`panel ${key} failed to load (${result.current.variant})`);
  });
  unmount();
};
