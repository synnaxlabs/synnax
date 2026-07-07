// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, createTestClient, DataType } from "@synnaxlabs/client";
import { Channel as PChannel } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Tree } from "@/platform/tree";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { createConsoleWrapper, createTestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const useDelete = Tree.createUseDelete({
  type: "Channel",
  query: PChannel.useDelete,
  convertKey: Number,
});

const DeleteHarness = ({
  props,
  label = "delete",
}: {
  props: Tree.ContextMenuProps;
  label?: string;
}): ReactElement => {
  const del = useDelete(props);
  return <button onClick={del}>{label}</button>;
};
DeleteHarness.displayName = "DeleteHarness";

const createChannel = async () =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

interface HarnessEntry {
  ch: channel.Channel;
  label?: string;
}

const setup = async (...entries: HarnessEntry[]) => {
  const store = await createTestStore();
  const createProps = (ch: channel.Channel): Tree.ContextMenuProps => {
    const otgID = channel.ontologyID(ch.key);
    return {
      ...createBaseProps({ client, store }),
      selection: createSelection({ ids: [otgID] }),
      state: createState([createResource(otgID, ch.name)]),
    };
  };
  const { wrapper: Console } = await createConsoleWrapper({ client, store });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>{children}</Console>
  );
  render(
    <>
      {entries.map(({ ch, label }) => (
        <DeleteHarness key={ch.key} props={createProps(ch)} label={label} />
      ))}
      <Modals.Stack />
    </>,
    { wrapper: Wrapper },
  );
};

const channelExists = async (key: channel.Key): Promise<boolean> => {
  try {
    await client.channels.retrieve(key);
    return true;
  } catch {
    return false;
  }
};

describe("createUseDelete", () => {
  it("should delete the selected resource after the user confirms", async () => {
    const ch = await createChannel();
    await setup({ ch });
    fireEvent.click(screen.getByText("delete"));
    await waitFor(() =>
      expect(
        screen.getByText(`Are you sure you want to delete ${ch.name}?`),
      ).toBeTruthy(),
    );
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await channelExists(ch.key)).toBe(false));
  });

  it("should leave the resource in place when the user cancels", async () => {
    const ch = await createChannel();
    const control = await createChannel();
    await setup(
      { ch, label: "delete target" },
      { ch: control, label: "delete control" },
    );
    fireEvent.click(screen.getByText("delete target"));
    await waitFor(() =>
      expect(
        screen.getByText(`Are you sure you want to delete ${ch.name}?`),
      ).toBeTruthy(),
    );
    fireEvent.click(findButton("Cancel"));
    await waitFor(() =>
      expect(
        screen.queryByText(`Are you sure you want to delete ${ch.name}?`),
      ).toBeNull(),
    );
    // A confirmed deletion through the same mutation path settles any erroneous
    // in-flight delete of the canceled target before the absence assert.
    fireEvent.click(screen.getByText("delete control"));
    await waitFor(() =>
      expect(
        screen.getByText(`Are you sure you want to delete ${control.name}?`),
      ).toBeTruthy(),
    );
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await channelExists(control.key)).toBe(false));
    expect(await channelExists(ch.key)).toBe(true);
  });
});
