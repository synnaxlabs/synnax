// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { Export } from "@/platform/export";
import { Import } from "@/platform/import";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import {
  createBaseProps,
  createSelection,
  createState,
  createTreeItems,
} from "@/platform/tree/testutil";
import { createConsoleWrapper, createTestStore, type TestStore } from "@/testutil";

export interface RenderTreeContextMenuOptions {
  client: Client;
  /** Resources backing the tree state. */
  resources: ontology.Resource[];
  /** The selected ids; defaults to every resource. */
  selected?: ontology.ID[];
  /** The tree parent of the selection; defaults to the root. */
  parentID?: ontology.ID;
  store?: TestStore;
  baseOverrides?: Partial<Tree.BaseProps>;
  stateOverrides?: Partial<Tree.TreeState>;
}

export interface TreeContextMenuHandle {
  store: TestStore;
  props: Tree.ContextMenuProps;
}

/**
 * Renders a domain's TreeContextMenu against the full console provider stack with a
 * live modal stack, backed by the given resources and selection. Returns the store and
 * the exact props handed to the menu so specs can re-invoke handlers or inspect state.
 */
export const renderTreeContextMenu = async (
  TreeContextMenu: Tree.ContextMenu,
  {
    client,
    resources,
    selected,
    parentID,
    store,
    baseOverrides = {},
    stateOverrides = {},
  }: RenderTreeContextMenuOptions,
): Promise<TreeContextMenuHandle> => {
  const resolvedStore = store ?? (await createTestStore());
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({
      client,
      store: resolvedStore,
      overrides: baseOverrides,
    }),
    selection: createSelection({
      ids: selected ?? resources.map((r) => r.id),
      parentID,
    }),
    state: createState(resources, stateOverrides),
  };
  const { wrapper } = await createConsoleWrapper({ client, store: resolvedStore });
  render(
    <Import.FileIngestersProvider fileIngesters={{}}>
      <Export.ExtractorsProvider extractors={{}}>
        <TreeContextMenu {...props} />
        <Modals.Stack />
      </Export.ExtractorsProvider>
    </Import.FileIngestersProvider>,
    { wrapper },
  );
  return { store: resolvedStore, props };
};

export interface RenderToolbarOptions {
  client: Client;
  /** Real tree items keyed by resource type; other types fall back to Tree.DEFAULT_ITEM. */
  items?: Tree.Items;
}

export interface ToolbarHandle {
  store: TestStore;
}

/**
 * Renders a domain's navdrawer toolbar content inside the full console provider stack
 * with haul + ontology services and a live modal stack, so the toolbar's resource tree
 * and its actions run against the real cluster.
 */
export const renderToolbar = async (
  content: ReactNode,
  { client, items }: RenderToolbarOptions,
): Promise<ToolbarHandle> => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Haul.Provider>
        <Tree.Provider items={createTreeItems(items)}>{children}</Tree.Provider>
      </Haul.Provider>
    </Console>
  );
  Wrapper.displayName = "ToolbarWrapper";
  render(
    <>
      {content}
      <Modals.Stack />
    </>,
    { wrapper: Wrapper },
  );
  return { store };
};

/** Returns the tree row (`.pluto-tree__item`) whose label matches text. */
export const getTreeRow = (text: string): HTMLElement => {
  const row = screen.getByText(text).closest<HTMLElement>(".pluto-tree__item");
  if (row == null) throw new Error(`tree row for ${text} not found`);
  return row;
};

/**
 * Clicks the expansion caret on the tree row matching text, loading its children.
 * Tree groups lazy-load their children only once expanded.
 */
export const expandTreeRow = (text: string): void => {
  const caret = getTreeRow(text).querySelector(".pluto-tree__expansion-indicator");
  if (caret == null) throw new Error(`expansion indicator for ${text} not found`);
  fireEvent.click(caret);
};

/**
 * Finds the button containing text inside the open modal dialog. Context menus render
 * items as buttons too, so an unscoped text query can hit the menu item that opened
 * the modal instead of the modal's own action.
 */
export const findModalButton = (text: string): HTMLButtonElement => {
  const dialog = screen.getByRole("dialog");
  const btn = within(dialog)
    .getAllByText(text)
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (btn == null) throw new Error(`modal button with text ${text} not found`);
  return btn;
};
