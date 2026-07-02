// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, createTestClient, DataType, ontology } from "@synnaxlabs/client";
import { Channel as PChannel, Text } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Ontology } from "@/platform/ontology";
import {
  buildBaseProps,
  buildResource,
  buildSelection,
  buildState,
} from "@/platform/ontology/testutil";
import { createConsoleWrapper, createTestStore } from "@/testutil";

const client = createTestClient();

const TIMEOUT = { timeout: 5000 };

const useRename = Ontology.createUseRename({
  query: PChannel.useRename,
  ontologyID: channel.ontologyID,
  convertKey: Number,
});

interface HarnessProps {
  props: Ontology.TreeContextMenuProps;
  itemID: string;
  name: string;
}

const RenameHarness = ({ props, itemID, name }: HarnessProps): ReactElement => {
  const rename = useRename(props);
  return (
    <>
      <button onClick={rename}>rename</button>
      <Text.MaybeEditable id={itemID} value={name} onChange={() => {}} />
    </>
  );
};
RenameHarness.displayName = "RenameHarness";

const createChannel = async () =>
  await client.channels.create({
    name: id.create(),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const setup = async (ch: channel.Channel): Promise<string> => {
  const otgID = channel.ontologyID(ch.key);
  const itemID = ontology.idToString(otgID);
  const store = await createTestStore();
  const props: Ontology.TreeContextMenuProps = {
    ...buildBaseProps({ client, store }),
    selection: buildSelection({ ids: [otgID] }),
    state: buildState([buildResource(otgID, ch.name)]),
  };
  const { wrapper: Console } = await createConsoleWrapper({ client, store });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>{children}</Console>
  );
  render(<RenameHarness props={props} itemID={itemID} name={ch.name} />, {
    wrapper: Wrapper,
  });
  return itemID;
};

const editableEl = (itemID: string): HTMLElement => {
  const el = document.querySelector<HTMLElement>(
    `[id="${itemID}"].pluto-text--editable`,
  );
  if (el == null) throw new Error(`editable element ${itemID} not found`);
  return el;
};

describe("createUseRename", () => {
  it("should start editing the resource's tree item when invoked", async () => {
    const ch = await createChannel();
    const itemID = await setup(ch);
    fireEvent.click(screen.getByText("rename"));
    await waitFor(() =>
      expect(editableEl(itemID).getAttribute("contenteditable")).toBe("true"),
    );
  });

  it("should rename the resource on the cluster once the edit is committed", async () => {
    const ch = await createChannel();
    const itemID = await setup(ch);
    const newName = id.create();
    fireEvent.click(screen.getByText("rename"));
    const el = editableEl(itemID);
    await waitFor(() => expect(el.getAttribute("contenteditable")).toBe("true"));
    await act(async () => {
      el.textContent = newName;
      fireEvent.keyDown(el, { key: "Enter" });
    });
    await waitFor(async () => {
      const renamed = await client.channels.retrieve(ch.key);
      expect(renamed.name).toBe(newName);
    }, TIMEOUT);
  });

  it("should not rename when the edit is escaped", async () => {
    const ch = await createChannel();
    const itemID = await setup(ch);
    fireEvent.click(screen.getByText("rename"));
    const el = editableEl(itemID);
    await waitFor(() => expect(el.getAttribute("contenteditable")).toBe("true"));
    await act(async () => {
      el.textContent = "should-not-apply";
      fireEvent.keyDown(el, { key: "Escape" });
    });
    await waitFor(() => expect(el.getAttribute("contenteditable")).toBe("false"));
    const unchanged = await client.channels.retrieve(ch.key);
    expect(unchanged.name).toBe(ch.name);
  });
});
