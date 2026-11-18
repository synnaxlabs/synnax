// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";
import { uniqueName } from "@/testutil";

const client = createTestClient();

/**
 * @returns a message of float fields, as a source reads it. Each field is named by a
 * key of channels and bound to its value.
 * @param index - The name of a timestamp field to read as the index.
 */
const createBound = (channels: Record<string, number>, index?: string) => {
  const fields = Object.keys(channels).map((name) =>
    icd.jsonFieldZ.parse({
      key: uuid.create(),
      name,
      pointer: `/${name}`,
      ...(name === index
        ? { type: "timestamp", timeFormat: "unix_ms" }
        : { type: "float" }),
    }),
  );
  const message: icd.Message = {
    key: uuid.create(),
    name: "Status",
    payload: { format: "json", fields },
  };
  const bindings = fields.map(({ key, name }) => ({
    field: key,
    channel: channels[name],
    dataType: "",
  }));
  return {
    source: {
      icd: uuid.create(),
      disabled: false,
      messages: [
        {
          message: message.key,
          index: fields.find(({ name }) => name === index)?.key,
          bindings,
        },
      ],
    },
    read: 0,
    message,
    fields,
  };
};

const createIndex = async (name: string) =>
  await client.channels.create({ name, dataType: "timestamp", isIndex: true });

describe("planMessage", () => {
  it("should plan a new channel for a field whose channel was deleted", async () => {
    const prefix = uniqueName("src");
    const index = await createIndex(`${prefix}_old_time`);
    const old = await client.channels.create({
      name: `${prefix}_old`,
      dataType: "float64",
      index: index.key,
    });
    await client.channels.delete(old.key);
    const bound = createBound({ temp: old.key });
    const plan = await ICD.planMessage(client, Task.createNamer(client), bound, prefix);
    expect(plan.fields[0].replaces).toBe(0);
    await ICD.applyMessage(client, bound, plan);
    const key = bound.source.messages[0].bindings[0].channel;
    expect(key).not.toBe(old.key);
    expect((await client.channels.retrieve(key)).name).toBe(`${prefix}_temp`);
  });

  it("should replace every channel on an index that another message writes", async () => {
    const prefix = uniqueName("src");
    const index = await createIndex(`${prefix}_time`);
    const createStored = async (name: string) =>
      await client.channels.create({
        name: `${prefix}_${name}`,
        dataType: "float64",
        index: index.key,
      });
    const shared = await createStored("temp");
    const stored = await createStored("pressure");
    const onIndex = await createStored("flow");
    const bound = createBound(
      { time: index.key, temp: 0, pressure: 0, flow: onIndex.key },
      "time",
    );
    const props = {
      index: index.key,
      channels: { "/temp": shared.key, "/pressure": stored.key },
    };
    const plan = await ICD.planMessage(
      client,
      Task.createNamer(client),
      bound,
      prefix,
      props,
      new Set([shared.key]),
    );
    await ICD.applyMessage(client, bound, plan);
    const [time, ...data] = await Promise.all(
      bound.source.messages[0].bindings.map(
        async ({ channel }) => await client.channels.retrieve(channel),
      ),
    );
    expect(time.key).not.toBe(index.key);
    expect(data.map(({ index }) => index)).toEqual([time.key, time.key, time.key]);
    expect(data.map(({ name }) => name)).toEqual([
      `${prefix}_temp_2`,
      `${prefix}_pressure`,
      `${prefix}_flow`,
    ]);
    expect((await client.channels.retrieve(shared.key)).name).toBe(`${prefix}_temp`);
  });
});
