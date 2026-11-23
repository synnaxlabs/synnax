// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel as channelClient, NotFoundError } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  createPlot,
  createReadTask,
  createVirtualChannel,
  openDelete,
  waitForDelete,
} from "@/feature/channel/testutil";
import { findLastButton } from "@/platform/modals/testutil";

const client = createTestClient();

const expectGone = async (retrieve: () => Promise<unknown>) =>
  await waitFor(async () => {
    await expect(retrieve()).rejects.toSatisfy((e) => NotFoundError.matches(e));
  });

describe("impact dialog claims check", () => {
  it("should list a status that the channel owns under Deleted with it", async () => {
    const ch = await createVirtualChannel(client);
    const name = uniqueName("alarm");
    await client.statuses.set({
      key: uniqueName("status"),
      name,
      variant: "warning",
      message: "Near its limit",
      time: TimeStamp.now(),
      owner: channelClient.resourceID(ch.key),
    });
    await openDelete(client, [ch]);
    const group = await screen.findByRole("region", { name: "Deleted with it" });
    await within(group).findByText(name);
  });

  it("should count a table that keeps a link to the channel", async () => {
    const ch = await createVirtualChannel(client);
    const proj = await client.projects.create({ name: uniqueName("proj") });
    await client.tables.create({
      project: proj.key,
      name: uniqueName("table"),
      columns: [{ key: "a" }],
      rows: [
        {
          key: "r",
          cells: [{ key: "a", config: { variant: "value", channel: ch.key } }],
        },
      ],
    });
    await openDelete(client, [ch]);
    await screen.findByText("1 more record keeps a link and does not change.");
    expect(
      screen.getByText(
        "Deletes the channel and its data. This action cannot be undone.",
      ),
    ).toBeTruthy();
  });

  it("should name the count of channels in a delete of two", async () => {
    const a = await createVirtualChannel(client);
    const b = await createVirtualChannel(client);
    await openDelete(client, [a, b]);
    await screen.findByText("Delete 2 channels");
    await screen.findByText(
      "Deletes the channels and their data. Nothing else uses them. " +
        "This action cannot be undone.",
    );
    await waitForDelete();
    fireEvent.click(findLastButton("Delete"));
    await expectGone(async () => await client.channels.retrieve(a.key));
    await expectGone(async () => await client.channels.retrieve(b.key));
  });

  it("should start every group but the first closed past 100 records", async () => {
    const ch = await createVirtualChannel(client);
    const task = await createReadTask(client, [ch.key]);
    const proj = await client.projects.create({ name: uniqueName("proj") });
    const plots = await client.lineplots.create(
      Array.from({ length: 101 }, (_, i) => ({
        project: proj.key,
        name: `${uniqueName("plot")}-${i}`,
        channels: { y1: [ch.key] },
      })),
    );
    await openDelete(client, [ch]);
    const blocks = await screen.findByRole("region", { name: "Blocks the delete" });
    await within(blocks).findByText(task.name);
    const loses = screen.getByRole("region", { name: "Loses the channel" });
    expect(within(loses).getByText("101")).toBeTruthy();
    for (const p of plots.slice(0, 8))
      expect(within(loses).queryByText(p.name)).toBeNull();
  });

  it("should order the records of a group by name", async () => {
    const ch = await createVirtualChannel(client);
    const prefix = uniqueName("plot");
    const names = ["e", "c", "a", "d", "b", "f"].map((s) => `${prefix}-${s}`);
    for (const n of names) await createPlot(client, ch.key, n);
    await openDelete(client, [ch]);
    const loses = await screen.findByRole("region", { name: "Loses the channel" });
    await within(loses).findByText(names[0]);
    const shown = within(loses)
      .getAllByText(new RegExp(`^${prefix}-`))
      .map((el) => el.textContent);
    expect(shown).toEqual(names.toSorted());
  });

  it("should mark a record that a changed plan adds as New", async () => {
    const ch = await createVirtualChannel(client);
    await openDelete(client, [ch]);
    await screen.findByText(
      "Deletes the channel and its data. Nothing else uses it. " +
        "This action cannot be undone.",
    );
    const plot = await createPlot(client, ch.key);
    await waitForDelete();
    fireEvent.click(findLastButton("Delete"));
    await screen.findByText("These records changed. Check them again.");
    const loses = await screen.findByRole("region", { name: "Loses the channel" });
    const name = await within(loses).findByText(plot.name);
    const item = name.parentElement;
    assertDefined(item);
    expect(within(item).getByText("New")).toBeTruthy();
  });

  it("should keep Delete disabled while a pick plans again", async () => {
    const ch = await createVirtualChannel(client);
    const task = await createReadTask(client, [ch.key]);
    await openDelete(client, [ch]);
    const check = await screen.findByRole("checkbox", {
      name: `${task.name} Delete too`,
    });
    await waitForDelete();
    fireEvent.click(check);
    expect(findLastButton("Delete")).toHaveAttribute("aria-disabled", "true");
    await waitForDelete();
  });

  it("should lock Cancel, the close, and Escape while the delete applies", async () => {
    const ch = await createVirtualChannel(client);
    await openDelete(client, [ch]);
    await waitForDelete();
    fireEvent.click(findLastButton("Delete"));
    expect(findLastButton("Cancel")).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByLabelText("Close")).toHaveAttribute("aria-disabled", "true");
    act(() => {
      fireEvent.keyDown(window, { code: "Escape" });
      fireEvent.keyUp(window, { code: "Escape" });
    });
    expect(screen.getByRole("dialog")).toBeTruthy();
    await expectGone(async () => await client.channels.retrieve(ch.key));
  });

  it("should name the role's limit when the Core refuses the delete", async () => {
    const ch = await createVirtualChannel(client);
    await createReadTask(client, [ch.key]);
    const userClient = await createTestClientWithPolicy(client, {
      name: uniqueName("impact"),
      objects: [channelClient.resourceID(ch.key)],
      actions: ["retrieve", "delete"],
    });
    await openDelete(userClient, [ch]);
    const blocks = await screen.findByRole("region", { name: "Blocks the delete" });
    fireEvent.click(
      within(blocks).getByRole("checkbox", {
        name: "1 record you cannot see Delete too",
      }),
    );
    await waitForDelete();
    fireEvent.click(findLastButton("Delete"));
    await screen.findByText("Failed to delete channel");
    await screen.findByText(
      /^Your role cannot change a record that this delete changes\. Ask an admin for access, or cancel\. /,
    );
    expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
  });
});
