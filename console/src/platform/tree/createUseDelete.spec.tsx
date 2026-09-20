// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Channel as PChannel } from "@synnaxlabs/pluto";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
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

const createChannel = async () =>
  await client.channels.create({
    name: uniqueName("ch"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });

const setup = async (target: channel.Channel) => {
  const store = await createTestStore();
  const propsFor = (ch: channel.Channel): Tree.ContextMenuProps => {
    const otgID = channel.ontologyID(ch.key);
    return {
      ...createBaseProps({ client, store }),
      selection: createSelection({ ids: [otgID] }),
      state: createState([createResource(otgID, ch.name)]),
    };
  };
  const { wrapper: Console } = await createConsoleWrapper({ client, store });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
      <Modals.Stack />
    </Console>
  );
  const { result, rerender } = renderHook(
    (props: Tree.ContextMenuProps) => useDelete(props),
    { wrapper, initialProps: propsFor(target) },
  );
  return {
    /** Invokes the delete, opening its confirmation modal. */
    del: async () => {
      // The confirmation content suspends, and content that suspends inside a
      // synchronous act scope is discarded, so the invocation needs an awaited one.
      await act(async () => {
        result.current();
      });
    },
    /** Points the hook at another resource, as a changed tree selection would. */
    select: (ch: channel.Channel) => rerender(propsFor(ch)),
  };
};

const channelExists = async (key: channel.Key): Promise<boolean> => {
  try {
    await client.channels.retrieve(key);
    return true;
  } catch {
    return false;
  }
};

const confirmationFor = (ch: channel.Channel): string =>
  `Are you sure you want to delete ${ch.name}?`;

describe("createUseDelete", () => {
  it("should delete the selected resource after the user confirms", async () => {
    const ch = await createChannel();
    const { del } = await setup(ch);
    await del();
    await waitFor(() => expect(screen.getByText(confirmationFor(ch))).toBeTruthy());
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await channelExists(ch.key)).toBe(false));
  });

  it("should leave the resource in place when the user cancels", async () => {
    const ch = await createChannel();
    const control = await createChannel();
    const { del, select } = await setup(ch);
    await del();
    await waitFor(() => expect(screen.getByText(confirmationFor(ch))).toBeTruthy());
    fireEvent.click(findButton("Cancel"));
    await waitFor(() => expect(screen.queryByText(confirmationFor(ch))).toBeNull());
    // A confirmed deletion through the same mutation path settles any erroneous
    // in-flight delete of the canceled target before the absence assert.
    select(control);
    await del();
    await waitFor(() =>
      expect(screen.getByText(confirmationFor(control))).toBeTruthy(),
    );
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => expect(await channelExists(control.key)).toBe(false));
    expect(await channelExists(ch.key)).toBe(true);
  });
});
