// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { deep } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { type FormRef, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createChannel = async () =>
  (
    await client.channels.create({
      name: uniqueName("bind"),
      dataType: "float32",
      virtual: true,
    })
  ).key;

interface Entry {
  key: string;
  port: number;
  channel: number;
}

const CHANNELS = deep.path<{ config: { channels: Entry[] } }>().config.channels;

const MAPPED: Record<number, number> = {
  1: await createChannel(),
  3: await createChannel(),
};

/** Resolves like a device that maps ports 1 and 3 and nothing else. */
const resolveMapped = ({ port }: Entry) => ({ channel: MAPPED[port] ?? 0 });

const channelOf = (form: FormRef, key: string) =>
  form.current?.get(CHANNELS[key].channel).value;

const render = async (
  entries: Entry[],
  resolve: (entry: Entry) => Partial<Entry> | null,
  mode?: "preview",
) =>
  await renderInTaskFormWithClient(
    <Task.BindChannels<Entry> path={CHANNELS} resolve={resolve} />,
    {
      client,
      values: { config: { channels: entries } },
      mode,
    },
  );

describe("BindChannels", () => {
  it("should bind an entry to the channel the device maps for its port", async () => {
    const { form } = await render([{ key: "a", port: 1, channel: 0 }], resolveMapped);
    await waitFor(() => expect(channelOf(form, "a")).toBe(MAPPED[1]));
  });

  it("should not bind a channel the Core no longer holds", async () => {
    const gone = await createChannel();
    await client.channels.delete(gone);
    const { form } = await render(
      [
        { key: "a", port: 1, channel: 0 },
        { key: "b", port: 2, channel: 0 },
      ],
      ({ port }) => ({ channel: port === 1 ? MAPPED[1] : gone }),
    );
    await waitFor(() => expect(channelOf(form, "a")).toBe(MAPPED[1]));
    expect(channelOf(form, "b")).toBe(0);
  });

  it("should match an entry whose binding does not change", async () => {
    const match = (entry: Entry): Entry =>
      entry.port === 9 ? entry : { ...entry, port: 9 };
    const { form } = await renderInTaskFormWithClient(
      <Task.BindChannels<Entry>
        path={CHANNELS}
        resolve={() => ({ channel: MAPPED[1] })}
        match={match}
      />,
      {
        client,
        values: { config: { channels: [{ key: "a", port: 1, channel: MAPPED[1] }] } },
      },
    );
    await waitFor(() => expect(form.current?.get(CHANNELS.a.port).value).toBe(9));
    expect(channelOf(form, "a")).toBe(MAPPED[1]);
  });

  it("should leave every entry alone while the device is unknown", async () => {
    const { form } = await render([{ key: "a", port: 1, channel: 7 }], () => null);
    await act(async () => {});
    expect(channelOf(form, "a")).toBe(7);
  });

  it("should not unbind an entry the device has not mapped yet", async () => {
    const { form } = await render([{ key: "a", port: 2, channel: 7 }], resolveMapped);
    await act(async () => {});
    expect(channelOf(form, "a")).toBe(7);
  });

  it("should move an entry to the channel its new port maps", async () => {
    const { form } = await render([{ key: "a", port: 1, channel: 0 }], resolveMapped);
    await waitFor(() => expect(channelOf(form, "a")).toBe(MAPPED[1]));
    act(() => form.current?.set(CHANNELS.a.port, 3));
    await waitFor(() => expect(channelOf(form, "a")).toBe(MAPPED[3]));
  });

  it("should unbind an entry moved to a port the device maps nothing for", async () => {
    const { form } = await render([{ key: "a", port: 1, channel: 0 }], resolveMapped);
    await waitFor(() => expect(channelOf(form, "a")).toBe(MAPPED[1]));
    act(() => form.current?.set(CHANNELS.a.port, 2));
    await waitFor(() => expect(channelOf(form, "a")).toBe(0));
  });

  it("should bind a channel nested in an object with the zero rule at the leaf", async () => {
    interface Nested {
      key: string;
      port: number;
      channel: { channel: number; name: string };
    }
    const nested = deep.path<{ config: { channels: Nested[] } }>().config.channels;
    const resolve = ({ port, channel }: Nested) => ({
      channel: { ...channel, channel: port === 1 ? MAPPED[1] : 0 },
    });
    const { form } = await renderInTaskFormWithClient(
      <Task.BindChannels<Nested> path={nested} resolve={resolve} />,
      {
        client,
        values: {
          config: {
            channels: [
              { key: "a", port: 1, channel: { channel: 0, name: "n" } },
              { key: "b", port: 2, channel: { channel: 7, name: "m" } },
            ],
          },
        },
      },
    );
    const leaf = (key: string) => form.current?.get(nested[key].channel.channel).value;
    await waitFor(() => expect(leaf("a")).toBe(MAPPED[1]));
    expect(leaf("b")).toBe(7);
    act(() => form.current?.set(nested.a.port, 2));
    await waitFor(() => expect(leaf("a")).toBe(0));
    expect(form.current?.get(nested.a.channel.name).value).toBe("n");
  });

  it("should leave a preview form alone", async () => {
    const { form } = await render(
      [{ key: "a", port: 1, channel: 0 }],
      resolveMapped,
      "preview",
    );
    await act(async () => {});
    expect(channelOf(form, "a")).toBe(0);
  });
});
