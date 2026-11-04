// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, NotFoundError, ontology } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ICD } from "@/feature/icd";
import { client, createICD, ENGINE_CSV } from "@/feature/icd/testutil";
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

const renderMenu = async (icds: icd.ICD[]) => {
  const ids = icds.map(({ key }) => icd.ontologyID(key));
  const store = await createTestStore();
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids }),
    state: createState(icds.map(({ name }, i) => createResource(ids[i], name))),
  };
  const { wrapper } = await createConsoleWrapper({ client, store });
  const Menu = ICD.TREE_ITEMS.icd?.ContextMenu;
  if (Menu == null) throw new Error("ICD tree item has no context menu");
  const itemID = List.itemNameID(ontology.idToString(ids[0]));
  render(
    <>
      <Menu {...props} />
      <Text.MaybeEditable id={itemID} value={icds[0].name} onChange={() => {}} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { itemID };
};

describe("ICD tree", () => {
  it("should offer import only for a single ICD", async () => {
    const [a, b] = [await createICD(), await createICD()];
    await renderMenu([a, b]);
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Import")).toBeNull();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should rename the ICD on the Core", async () => {
    const ic = await createICD();
    const { itemID } = await renderMenu([ic]);
    fireEvent.click(await screen.findByText("Rename"));
    const el = await awaitTextEditing(itemID);
    const name = uniqueName("renamed");
    await act(async () => commitTextEdit(el, name));
    await waitFor(async () =>
      expect((await client.icds.retrieve({ key: ic.key })).name).toBe(name),
    );
  });

  it("should delete the ICD after confirmation", async () => {
    const ic = await createICD();
    await renderMenu([ic]);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${ic.name}?`);
    fireEvent.click(findLastButton("Delete"));
    await waitFor(
      async () =>
        await expect(client.icds.retrieve({ key: ic.key })).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        ),
    );
  });

  it("should import a picked file in the format its extension names", async () => {
    const picker = interceptFilePicker();
    const ic = await createICD();
    await renderMenu([ic]);
    fireEvent.click(await screen.findByText("Import"));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("engine.csv", ENGINE_CSV)]);
    await waitFor(async () => {
      const { messages } = await client.icds.retrieve({ key: ic.key });
      expect(messages.map(({ name }) => name)).toEqual(["Engine"]);
    });
  });
});
