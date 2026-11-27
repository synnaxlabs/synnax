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
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, render, screen } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Tree } from "@/platform/tree";
import { UsedBy } from "@/platform/used-by";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const createChannel = async (): Promise<channel.Channel> => {
  const ch = await client.channels.create({
    name: uniqueName("use"),
    dataType: DataType.FLOAT32,
    virtual: true,
  });
  onTestFinished(async () => await client.channels.delete(ch.key));
  return ch;
};

const renderPopover = async (uses: UsedBy.Use[]): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  await act(async () =>
    render(
      <Tree.Provider items={{}}>
        <UsedBy.Popover itemKey="item" kind="channel" uses={uses} anchor={null} />
      </Tree.Provider>,
      { wrapper },
    ),
  );
};

describe("UsedBy.Popover", () => {
  it("should count a resource that reads and writes the item once", async () => {
    const both = await createChannel();
    const id = channel.resourceID(both.key);
    await renderPopover([
      { id, direction: "read" },
      { id, direction: "write" },
    ]);
    expect(await screen.findByText("Used by 1")).toBeTruthy();
  });

  it("should count each resource that uses the item", async () => {
    const [reader, writer] = await Promise.all([createChannel(), createChannel()]);
    await renderPopover([
      { id: channel.resourceID(reader.key), direction: "read" },
      { id: channel.resourceID(writer.key), direction: "write" },
    ]);
    expect(await screen.findByText("Used by 2")).toBeTruthy();
  });

  it("should show no count when nothing uses the item", async () => {
    await renderPopover([]);
    expect(screen.queryByText(/^Used by \d/)).toBeNull();
  });
});
