// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library, NotFoundError, ontology } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Library } from "@/feature/library";
import { client, createLibrary, ENGINE_CSV } from "@/feature/library/testutil";
import { Modals } from "@/platform/modals";
import { findLastButton } from "@/platform/modals/testutil";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import {
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  createTestStore,
  fakePickedFile,
  interceptFilePicker,
  uniqueName,
} from "@/testutil";

const renderMenu = async (libraries: library.Library[]) => {
  const ids = libraries.map(({ key }) => library.ontologyID(key));
  const store = await createTestStore();
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids }),
    state: createState(libraries.map(({ name }, i) => createResource(ids[i], name))),
  };
  const { wrapper } = await createConsoleWrapper({ client, store });
  const Menu = Library.TREE_ITEMS.library.ContextMenu;
  if (Menu == null) throw new Error("library tree item has no context menu");
  const itemID = List.itemNameID(ontology.idToString(ids[0]));
  render(
    <>
      <Menu {...props} />
      <Text.MaybeEditable id={itemID} value={libraries[0].name} onChange={() => {}} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { itemID };
};

describe("Library tree", () => {
  it("should offer import only for a single library", async () => {
    const [a, b] = [await createLibrary(), await createLibrary()];
    await renderMenu([a, b]);
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Import")).toBeNull();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should rename the library on the Core", async () => {
    const lib = await createLibrary();
    const { itemID } = await renderMenu([lib]);
    fireEvent.click(await screen.findByText("Rename"));
    const el = await awaitTextEditing(itemID);
    const name = uniqueName("renamed");
    await act(async () => commitTextEdit(el, name));
    await waitFor(async () =>
      expect((await client.libraries.retrieve({ key: lib.key })).name).toBe(name),
    );
  });

  it("should delete the library after confirmation", async () => {
    const lib = await createLibrary();
    await renderMenu([lib]);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${lib.name}?`);
    fireEvent.click(findLastButton("Delete"));
    await waitFor(
      async () =>
        await expect(client.libraries.retrieve({ key: lib.key })).rejects.toSatisfy(
          (e) => NotFoundError.matches(e),
        ),
    );
  });

  it("should import a picked file in the format its extension names", async () => {
    const picker = interceptFilePicker();
    const lib = await createLibrary();
    await renderMenu([lib]);
    fireEvent.click(await screen.findByText("Import"));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("engine.csv", ENGINE_CSV)]);
    await waitFor(async () => {
      const { entries } = await client.libraries.retrieve({ key: lib.key });
      expect(entries.map(({ name }) => name)).toEqual(["Engine"]);
    });
  });
});
