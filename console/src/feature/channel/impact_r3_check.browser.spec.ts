// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createReadTask,
  createVirtualChannel,
  openDelete,
  waitForDelete,
} from "@/feature/channel/testutil";
import { findLastButton } from "@/platform/modals/testutil";

const client = createTestClient();

const CHANGED = "These records changed. Check them again.";

const findPickOf = async (name: string): Promise<HTMLElement> =>
  await screen.findByRole("checkbox", { name: `${name} Delete too` });

describe("impact dialog round 3 checks", () => {
  // A pick is consent for the record it names. Once a plan stops listing that record,
  // the record must come back unchecked.
  it("should show a record unchecked when the pick that brings it is made again", async () => {
    const index = await client.channels.create({
      name: uniqueName("index"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const data = await client.channels.create({
      name: uniqueName("data"),
      dataType: DataType.UINT8,
      index: index.key,
    });
    const task = await createReadTask(client, [data.key]);
    await openDelete(client, [index]);
    fireEvent.click(await findPickOf(data.name));
    fireEvent.click(await findPickOf(task.name));
    await waitForDelete();
    fireEvent.click(await findPickOf(data.name));
    await waitFor(() => expect(screen.queryByText(task.name)).toBeNull());
    await waitForDelete();
    fireEvent.click(await findPickOf(data.name));
    expect(await findPickOf(task.name)).not.toBeChecked();
  });

  // D-331: the hidden pick goes out only while a hidden record blocks. A check made for
  // one hidden blocker must not carry over to a hidden blocker that came later.
  it("should show a later hidden blocker unchecked", async () => {
    const ch = await createVirtualChannel(client);
    const first = await createReadTask(client, [ch.key]);
    const userClient = await createTestClientWithPolicy(client, {
      name: uniqueName("impact"),
      objects: [channel.resourceID(ch.key)],
      actions: ["retrieve", "delete"],
    });
    await openDelete(userClient, [ch]);
    fireEvent.click(await screen.findByRole("checkbox", { name: /you cannot see/ }));
    await waitForDelete();
    await client.tasks.delete(first.key);
    fireEvent.click(findLastButton("Delete"));
    await screen.findByText(CHANGED);
    await waitFor(() =>
      expect(screen.queryByRole("checkbox", { name: /you cannot see/ })).toBeNull(),
    );
    await waitForDelete();
    await createReadTask(client, [ch.key]);
    fireEvent.click(findLastButton("Delete"));
    const later = await screen.findByRole("checkbox", { name: /you cannot see/ });
    expect(later).not.toBeChecked();
    expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
  });
});
