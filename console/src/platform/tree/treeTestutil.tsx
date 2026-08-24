// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax } from "@synnaxlabs/client";
import { Haul, Triggers } from "@synnaxlabs/pluto";
import {
  act,
  fireEvent,
  render,
  type RenderResult,
  screen,
} from "@testing-library/react";
import { type ReactNode } from "react";

import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { createConsoleWrapper, selectTestProject, type TestStore } from "@/testutil";

export interface RenderOntologyTreeOptions {
  client: Synnax;
  root: ontology.ID | null;
  /** Real item overrides; every other resource type falls back to Tree.DefaultItem. */
  items?: Tree.Items;
  /** Extra siblings rendered alongside the tree, e.g. CaptureStatuses. */
  extra?: ReactNode;
  /** The client that creates the session's project; defaults to the rendering one. */
  projectClient?: Synnax;
}

export interface OntologyTreeHandle extends RenderResult {
  store: TestStore;
}

/**
 * Renders the real Tree.Tree against the live Core inside the full console
 * provider stack, with a mounted modal stack so context-menu flows can open prompts.
 * The Triggers provider is included so held-modifier interactions (control-click
 * multi-select) behave as they do in the app. An active project is established so
 * onSelect handlers that place a tab have a project to create panels under.
 */
export const renderOntologyTree = async ({
  client,
  root,
  items = {},
  extra,
  projectClient = client,
}: RenderOntologyTreeOptions): Promise<OntologyTreeHandle> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  await selectTestProject(store, projectClient);
  const rendered = render(
    <Triggers.Provider>
      <Haul.Provider>
        <Tree.Provider items={items}>
          <Tree.Tree root={root} />
          <Modals.Stack />
        </Tree.Provider>
      </Haul.Provider>
      {extra}
    </Triggers.Provider>,
    { wrapper },
  );
  return { ...rendered, store };
};

const SCROLL_STEP = 200;

/**
 * Finds the tree label matching text, scrolling the tree when the row sits outside the
 * window. The tree keeps only the rows near the window mounted, so a label further down
 * reaches the DOM the same way a user reaches it. Leaves the tree scrolled where the
 * label was found, and back at the top when there is none.
 */
// Sweeps from the top: a previous lookup can leave the tree scrolled past the row.
const sweepTree = async (text: string): Promise<HTMLElement | null> => {
  const tree = document.querySelector<HTMLElement>(".pluto-tree");
  const rows = tree?.querySelector<HTMLElement>(".pluto-list__virtualizer");
  if (tree == null || rows == null) return null;
  const total = parseInt(rows.style.minHeight, 10);
  for (let offset = 0; offset <= total; offset += SCROLL_STEP) {
    await act(async () => {
      tree.scrollTop = offset;
      fireEvent.scroll(tree);
    });
    const label = screen.queryByText(text);
    if (label != null) return label;
  }
  return null;
};

const findTreeLabel = async (text: string): Promise<HTMLElement> => {
  const mounted = screen.queryByText(text);
  if (mounted != null) return mounted;
  const swept = await sweepTree(text);
  if (swept != null) return swept;
  // Absent from the data rather than the window: wait for it to load, then sweep again.
  try {
    return await screen.findByText(text);
  } catch (err) {
    const retried = await sweepTree(text);
    if (retried != null) return retried;
    throw err instanceof Error ? err : new Error(String(err), { cause: err });
  }
};

/** Finds the rendered tree row containing the given text. */
export const findTreeRow = async (text: string): Promise<Element> => {
  const row = (await findTreeLabel(text)).closest(".pluto-tree__item");
  if (row == null) throw new Error(`tree row for ${text} not found`);
  return row;
};

/** Right-clicks the tree row containing text, opening its service context menu. A
 * menu that suspends must mount inside an async act: a suspension mounted under
 * sync act never retries when its promise resolves, so the menu stays empty. */
export const openTreeRowContextMenu = async (text: string): Promise<void> => {
  const row = await findTreeRow(text);
  await act(async () => {
    fireEvent.contextMenu(row);
  });
};
