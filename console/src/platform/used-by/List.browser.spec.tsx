// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  DataType,
  ranger,
  type Synnax as Client,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  fireEvent,
  render,
  type RenderOptions,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { Tree } from "@/platform/tree";
import { UsedBy } from "@/platform/used-by";
import { createConsoleWrapper, createTestClientWithGrants } from "@/testutil";

const client = createTestClient();

const createChannel = async (name = uniqueName("use")): Promise<channel.Channel> => {
  const ch = await client.channels.create({
    name,
    dataType: DataType.FLOAT32,
    virtual: true,
  });
  onTestFinished(async () => await client.channels.delete(ch.key));
  return ch;
};

const createRange = async (name = uniqueName("use")): Promise<ranger.Range> => {
  const r = await client.ranges.create({
    name,
    timeRange: new TimeRange(TimeStamp.now(), TimeStamp.now().add(TimeSpan.SECOND)),
  });
  onTestFinished(async () => await client.ranges.delete(r.key));
  return r;
};

const read = (ch: channel.Channel): UsedBy.Use => ({
  id: channel.resourceID(ch.key),
  direction: "read",
});

const write = (ch: channel.Channel): UsedBy.Use => ({
  id: channel.resourceID(ch.key),
  direction: "write",
});

interface RenderListOptions extends Pick<RenderOptions, "onCaughtError"> {
  /** The tree items that open each type of use. */
  items?: Tree.Items;
  /** The client that reads the uses. */
  as?: Client;
}

/** Renders the list, and returns a function that renders it again with other props. */
const renderList = async (
  props: UsedBy.ListProps,
  { items = {}, as = client, onCaughtError }: RenderListOptions = {},
): Promise<(next: UsedBy.ListProps) => Promise<void>> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  const tree = (p: UsedBy.ListProps) => (
    <Tree.Provider items={items}>
      <UsedBy.List {...p} />
    </Tree.Provider>
  );
  // A suspension mounted under a sync act never retries when its promise resolves.
  const { rerender } = await act(async () =>
    render(tree(props), { wrapper, onCaughtError }),
  );
  return async (next) => {
    await act(async () => rerender(tree(next)));
  };
};

const headings = (): (string | null)[] =>
  screen.getAllByRole("group").map((g) => g.getAttribute("aria-label"));

describe("UsedBy.List", () => {
  it("should list the uses with no direction by name under no heading, ranges included", async () => {
    const base = uniqueName("use");
    const b = await createRange(`${base}_b`);
    const a = await createChannel(`${base}_a`);
    const c = await createRange(`${base}_c`);
    await renderList({
      kind: "secret",
      uses: [
        { id: ranger.resourceID(c.key) },
        { id: channel.resourceID(a.key) },
        { id: ranger.resourceID(b.key) },
      ],
    });
    await screen.findByText(a.name);
    const listed = screen.getAllByText((text) => text.startsWith(base));
    expect(listed.map((el) => el.textContent)).toEqual([a.name, b.name, c.name]);
    expect(headings()).toEqual([null]);
  });

  it("should list the writes, then the reads, each under a heading with its count", async () => {
    const [reader, writer, other] = await Promise.all([
      createChannel(),
      createChannel(),
      createChannel(),
    ]);
    await renderList({
      kind: "channel",
      uses: [read(reader), write(writer), read(other)],
    });
    await screen.findByText(reader.name);
    expect(headings()).toEqual(["Written by", "Read by"]);
    const written = within(screen.getByRole("group", { name: "Written by" }));
    expect(written.getByText("1")).toBeTruthy();
    expect(written.getByText(writer.name)).toBeTruthy();
    const reads = within(screen.getByRole("group", { name: "Read by" }));
    expect(reads.getByText("2")).toBeTruthy();
    expect(reads.getByText(reader.name)).toBeTruthy();
    expect(reads.getByText(other.name)).toBeTruthy();
  });

  it("should leave out the heading of a direction that no use has", async () => {
    const reader = await createChannel();
    await renderList({ kind: "channel", uses: [read(reader)] });
    await screen.findByText(reader.name);
    expect(headings()).toEqual(["Read by"]);
  });

  it("should list a resource that writes and reads the item under both headings", async () => {
    const both = await createChannel();
    await renderList({ kind: "channel", uses: [read(both), write(both)] });
    await screen.findAllByText(both.name);
    for (const name of ["Written by", "Read by"])
      expect(
        within(screen.getByRole("group", { name })).getByText(both.name),
      ).toBeTruthy();
  });

  it("should leave out a use whose resource was deleted after the uses were read", async () => {
    const [kept, deleted] = await Promise.all([createChannel(), createChannel()]);
    await client.channels.delete(deleted.key);
    await renderList({ kind: "channel", uses: [read(kept), read(deleted)] });
    await screen.findByText(kept.name);
    const reads = within(screen.getByRole("group", { name: "Read by" }));
    expect(reads.getByText("1")).toBeTruthy();
    expect(screen.queryByText(deleted.name)).toBeNull();
  });

  it("should say that nothing uses the item when every use was deleted", async () => {
    const deleted = await createRange();
    await client.ranges.delete(deleted.key);
    await renderList({
      kind: "enumeration",
      uses: [{ id: ranger.resourceID(deleted.key) }],
    });
    expect(await screen.findByText("Nothing uses this enumeration.")).toBeTruthy();
  });

  it("should show the type of a use and open it through its tree item", async () => {
    const onOpen = vi.fn();
    const ch = await createChannel();
    await renderList(
      { kind: "range", uses: [read(ch)] },
      {
        items: {
          channel: Tree.createItem({ typeName: "Channel", useOnSelect: () => onOpen }),
        },
      },
    );
    fireEvent.click(await screen.findByText(ch.name));
    expect(screen.getByText("Channel")).toBeTruthy();
    expect(onOpen).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ id: channel.resourceID(ch.key), name: ch.name }),
    );
  });

  it("should offer no click on a use whose tree item opens nothing", async () => {
    const ch = await createChannel();
    await renderList(
      { kind: "range", uses: [read(ch)] },
      { items: { channel: Tree.createItem({ typeName: "Channel" }) } },
    );
    await screen.findByText(ch.name);
    expect(screen.queryByRole("button", { name: new RegExp(ch.name) })).toBeNull();
  });

  it("should run the select of a type in place of its tree item", async () => {
    const onOpen = vi.fn();
    const select = vi.fn();
    const onSelect = vi.fn();
    const ch = await createChannel();
    await renderList(
      { kind: "range", uses: [read(ch)], selects: { channel: select }, onSelect },
      { items: { channel: Tree.createItem({ useOnSelect: () => onOpen }) } },
    );
    fireEvent.click(await screen.findByText(ch.name));
    expect(select).toHaveBeenCalledExactlyOnceWith(String(ch.key));
    expect(onSelect).toHaveBeenCalledOnce();
    expect(onOpen).not.toHaveBeenCalled();
  });

  it("should say why the read of the uses failed", async () => {
    const r = await createRange();
    const onCaughtError = vi.fn();
    await renderList(
      { kind: "channel", uses: [{ id: ranger.resourceID(r.key) }] },
      { as: await createTestClientWithGrants(client), onCaughtError },
    );
    expect(await screen.findByText("Failed to load the uses")).toBeTruthy();
    expect(screen.getByText(/cannot retrieve range.*access denied/)).toBeTruthy();
    expect(onCaughtError).toHaveBeenCalledOnce();
  });

  it("should read again when the uses change after a failed read", async () => {
    const [denied, granted] = await Promise.all([createRange(), createRange()]);
    const limited = await createTestClientWithGrants(client, {
      retrieve: [ranger.resourceID(granted.key)],
    });
    const rerender = await renderList(
      { kind: "channel", uses: [{ id: ranger.resourceID(denied.key) }] },
      { as: limited, onCaughtError: vi.fn() },
    );
    await screen.findByText("Failed to load the uses");
    await rerender({ kind: "channel", uses: [{ id: ranger.resourceID(granted.key) }] });
    expect(await screen.findByText(granted.name)).toBeTruthy();
    expect(screen.queryByText("Failed to load the uses")).toBeNull();
  });

  it("should read the uses again on Retry and keep the focus in the list", async () => {
    const r = await createRange();
    const limited = await createTestClientWithGrants(client);
    await renderList(
      { kind: "channel", uses: [{ id: ranger.resourceID(r.key) }] },
      { as: limited, onCaughtError: vi.fn() },
    );
    // The read that a click starts suspends, which needs an async act.
    const retry = async () => {
      const button = await screen.findByRole("button", { name: "Retry" });
      await act(async () => {
        fireEvent.click(button);
      });
    };
    await retry();
    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByRole("button", { name: "Retry" }),
      ),
    );
    const { role } = await limited.users.current.retrieve();
    await client.access.policies.create({
      name: uniqueName("policy"),
      role,
      objects: [ranger.resourceID(r.key)],
      actions: ["retrieve"],
    });
    await retry();
    expect(await screen.findByText(r.name)).toBeTruthy();
    expect(document.activeElement).not.toBe(document.body);
  });
});
