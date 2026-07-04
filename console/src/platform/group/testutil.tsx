// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax as Client } from "@synnaxlabs/client";
import { Haul, Triggers } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";

import { Ontology } from "@/platform/ontology";
import { createServices } from "@/platform/ontology/testutil";
import { createConsoleWrapper } from "@/testutil";

export interface RenderOntologyTreeArgs {
  client: Client;
  root: ontology.ID;
  services?: Ontology.Services;
}

/**
 * Mounts the real ontology tree against the full console provider stack, backed by the
 * given client. The Triggers provider is included so held-modifier interactions
 * (control-click multi-select) behave as they do in the app.
 */
export const renderOntologyTree = async ({
  client,
  root,
  services = createServices(),
}: RenderOntologyTreeArgs) => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Triggers.Provider>
        <Haul.Provider>{children}</Haul.Provider>
      </Triggers.Provider>
    </Console>
  );
  return render(
    <Ontology.ServicesProvider services={services}>
      <Ontology.Tree root={root} />
    </Ontology.ServicesProvider>,
    { wrapper: Wrapper },
  );
};

/** Finds the rendered tree row containing the given item text. */
export const treeRow = (text: string): Element => {
  const row = screen.getByText(text).closest(".pluto-tree__item");
  if (row == null) throw new Error(`tree row for ${text} not found`);
  return row;
};

/** Expands the tree row containing the given item text via its caret. */
export const expandTreeRow = (text: string): void => {
  const caret = treeRow(text).querySelector(".pluto-tree__expansion-indicator");
  if (caret == null) throw new Error(`expansion indicator for ${text} not found`);
  fireEvent.click(caret);
};

/** Runs interactions while the Control key is held through the Triggers provider. */
export const withControlHeld = (run: () => void): void => {
  fireEvent.keyDown(window, { key: "Control", code: "ControlLeft" });
  try {
    run();
  } finally {
    fireEvent.keyUp(window, { key: "Control", code: "ControlLeft" });
  }
};
