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

const PATH = "config.sources.s";

const SOURCE = {
  key: "s",
  path: "/api",
  index: "ts",
  fields: [
    { key: "ts", pointer: "/time", channel: 0 },
    { key: "v", pointer: "/value", channel: 0 },
  ],
};

const READ: Record<string, Task.ReadChannelProps> = {
  "/api": { index: 4, channels: { "/value": 6 } },
};

const channelOf = (form: FormRef, key: string) =>
  form.current?.get<number>(`${PATH}.fields.${key}.channel`).value;

const render = async (read: Record<string, Task.ReadChannelProps> | null) =>
  await renderInTaskForm(
    <Task.BindReadChannels path={PATH} propertiesKeyField="path" read={read} />,
    { values: { config: { sources: [SOURCE] } } },
  );

describe("BindReadChannels", () => {
  it("should bind the index field to the index and a field to its pointer", async () => {
    const { form } = await render(READ);
    await waitFor(() => expect(channelOf(form, "v")).toBe(6));
    expect(channelOf(form, "ts")).toBe(4);
  });

  it("should leave every field alone while the device is unknown", async () => {
    const { form } = await render(null);
    await act(async () => {});
    expect(channelOf(form, "v")).toBe(0);
  });

  it("should follow the source to the channels its new key maps", async () => {
    const { form } = await render({
      ...READ,
      "/other": { index: 8, channels: { "/value": 9 } },
    });
    await waitFor(() => expect(channelOf(form, "v")).toBe(6));
    act(() => form.current?.set(`${PATH}.path`, "/other"));
    await waitFor(() => expect(channelOf(form, "v")).toBe(9));
    expect(channelOf(form, "ts")).toBe(8);
  });
});
