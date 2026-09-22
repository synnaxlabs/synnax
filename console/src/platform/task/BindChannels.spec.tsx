// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { type FormRef, renderInTaskForm } from "@/platform/task/testutil";

interface Entry {
  key: string;
  port: number;
  channel: number;
}

const PATH = "config.channels";

const MAPPED = { 1: 5, 3: 9 } as const satisfies Record<number, number>;

/** Resolves like a device that maps ports 1 and 3 and nothing else. */
const resolveMapped = ({ port }: Entry) => ({
  channel: port === 1 || port === 3 ? MAPPED[port] : 0,
});

const channelOf = (form: FormRef, key: string) =>
  form.current?.get<number>(`${PATH}.${key}.channel`).value;

const render = async (
  entries: Entry[],
  resolve: (entry: Entry) => Partial<Entry> | null,
  mode?: "preview",
) =>
  await renderInTaskForm(<Task.BindChannels<Entry> resolve={resolve} />, {
    values: { config: { channels: entries } },
    mode,
  });

describe("BindChannels", () => {
  it("should bind an entry to the channel the device maps for its port", async () => {
    const { form } = await render([{ key: "a", port: 1, channel: 0 }], resolveMapped);
    await waitFor(() => expect(channelOf(form, "a")).toBe(5));
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
    await waitFor(() => expect(channelOf(form, "a")).toBe(5));
    act(() => form.current?.set(`${PATH}.a.port`, 3));
    await waitFor(() => expect(channelOf(form, "a")).toBe(9));
  });

  it("should unbind an entry moved to a port the device maps nothing for", async () => {
    const { form } = await render([{ key: "a", port: 1, channel: 0 }], resolveMapped);
    await waitFor(() => expect(channelOf(form, "a")).toBe(5));
    act(() => form.current?.set(`${PATH}.a.port`, 2));
    await waitFor(() => expect(channelOf(form, "a")).toBe(0));
  });

  it("should bind a channel nested in an object with the zero rule at the leaf", async () => {
    interface Nested {
      key: string;
      port: number;
      channel: { channel: number; name: string };
    }
    const resolve = ({ port, channel }: Nested) => ({
      channel: { ...channel, channel: port === 1 ? 5 : 0 },
    });
    const { form } = await renderInTaskForm(
      <Task.BindChannels<Nested> resolve={resolve} />,
      {
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
    const leaf = (key: string) =>
      form.current?.get<number>(`${PATH}.${key}.channel.channel`).value;
    await waitFor(() => expect(leaf("a")).toBe(5));
    expect(leaf("b")).toBe(7);
    act(() => form.current?.set(`${PATH}.a.port`, 2));
    await waitFor(() => expect(leaf("a")).toBe(0));
    expect(form.current?.get<string>(`${PATH}.a.channel.name`).value).toBe("n");
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
