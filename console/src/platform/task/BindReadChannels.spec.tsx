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
      name: uniqueName("bind_read"),
      dataType: "float32",
      virtual: true,
    })
  ).key;

const [INDEX, VALUE, OTHER_INDEX, OTHER_VALUE] = await Promise.all(
  Array.from({ length: 4 }, createChannel),
);

interface Source {
  key: string;
  path: string;
  index: string;
  fields: Task.ReadChannelField[];
}

const source = deep.path<{ config: { sources: Source[] } }>().config.sources.s;

const field = (key: string, pointer: string): Task.ReadChannelField => ({
  key,
  pointer,
  channel: 0,
  name: "",
  dataType: "float64",
});

const SOURCE: Source = {
  key: "s",
  path: "/api",
  index: "ts",
  fields: [field("ts", "/time"), field("v", "/value")],
};

const READ: Record<string, Task.ReadChannelProps> = {
  "/api": { index: INDEX, channels: { "/value": VALUE } },
};

const channelOf = (form: FormRef, key: string) =>
  form.current?.get(source.fields[key].channel).value;

const render = async (read: Record<string, Task.ReadChannelProps> | null) =>
  await renderInTaskFormWithClient(
    <Task.BindReadChannels path={source} propertiesKey={source.path} read={read} />,
    { client, values: { config: { sources: [SOURCE] } } },
  );

describe("BindReadChannels", () => {
  it("should bind the index field to the index and a field to its pointer", async () => {
    const { form } = await render(READ);
    await waitFor(() => expect(channelOf(form, "v")).toBe(VALUE));
    expect(channelOf(form, "ts")).toBe(INDEX);
  });

  it("should leave every field alone while the device is unknown", async () => {
    const { form } = await render(null);
    await act(async () => {});
    expect(channelOf(form, "v")).toBe(0);
  });

  it("should follow the source to the channels its new key maps", async () => {
    const { form } = await render({
      ...READ,
      "/other": { index: OTHER_INDEX, channels: { "/value": OTHER_VALUE } },
    });
    await waitFor(() => expect(channelOf(form, "v")).toBe(VALUE));
    act(() => form.current?.set(source.path, "/other"));
    await waitFor(() => expect(channelOf(form, "v")).toBe(OTHER_VALUE));
    expect(channelOf(form, "ts")).toBe(OTHER_INDEX);
  });
});
