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
import { fireEvent, render, type RenderResult, screen } from "@testing-library/react";

import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { createConsoleWrapper, type TestStore } from "@/testutil";

export interface RenderOntologyTreeOptions {
  client: Synnax;
  root: ontology.ID | null;
  /** Real item overrides; every other resource type falls back to Tree.DefaultItem. */
  items?: Tree.Items;
}

export interface OntologyTreeHandle extends RenderResult {
  store: TestStore;
}

/**
 * Renders the real Tree.Tree against the live cluster inside the full console
 * provider stack, with a mounted modal stack so context-menu flows can open prompts.
 * The Triggers provider is included so held-modifier interactions (control-click
 * multi-select) behave as they do in the app.
 */
export const renderOntologyTree = async ({
  client,
  root,
  items = {},
}: RenderOntologyTreeOptions): Promise<OntologyTreeHandle> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const rendered = render(
    <Triggers.Provider>
      <Haul.Provider>
        <Tree.Provider items={items}>
          <Tree.Tree root={root} />
          <Modals.Stack />
        </Tree.Provider>
      </Haul.Provider>
    </Triggers.Provider>,
    { wrapper },
  );
  return { ...rendered, store };
};

/** Finds the rendered tree row containing the given text. */
export const findTreeRow = async (text: string): Promise<Element> => {
  const row = (await screen.findByText(text)).closest(".pluto-tree__item");
  if (row == null) throw new Error(`tree row for ${text} not found`);
  return row;
};

/** Right-clicks the tree row containing text, opening its service context menu. */
export const openTreeRowContextMenu = async (text: string): Promise<void> => {
  fireEvent.contextMenu(await findTreeRow(text));
};
