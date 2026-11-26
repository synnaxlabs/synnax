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
import { uniqueName } from "@synnaxlabs/x/testutil";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

const client = createTestClient();

/**
 * @returns a message of fields of one type, as a source reads it. Each field is named
 * by a key of channels and bound to its value.
 * @param index - The name of a timestamp field to read as the index.
 * @param field - The settings of each field but the index, such as its type.
 */
const createBound = (
  channels: Record<string, number>,
  index?: string,
  field: Record<string, unknown> = { type: "float" },
) => {
  const fields = Object.keys(channels).map((name) =>
    icd.jsonFieldZ.parse({
      key: uuid.create(),
      name,
      pointer: `/${name}`,
      ...(name === index ? { type: "timestamp", timeFormat: "unix_ms" } : field),
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

interface Kind {
  /** The name of the timestamp field that the kind reads as its index, if any. */
  index?: string;
  /** The names of its float fields. The match rule needs the first one. */
  fields: string[];
  /** The channel of each field by name. A field with none gets a new one. */
  channels?: Record<string, number>;
}

/** @returns a message for each kind, as one source reads them. */
const createKinds = (kinds: Kind[]) => {
  const source = {
    icd: uuid.create(),
    disabled: false,
    messages: [] as icd.ReadMessage[],
  };
  return kinds.map(({ index, fields: names, channels = {} }, read) => {
    const indexField =
      index == null
        ? undefined
        : icd.jsonFieldZ.parse({
            key: uuid.create(),
            name: index,
            pointer: `/${index}`,
            type: "timestamp",
            timeFormat: "unix_ms",
          });
    const fields = [
      ...(indexField == null ? [] : [indexField]),
      ...names.map((name) =>
        icd.jsonFieldZ.parse({
          key: uuid.create(),
          name,
          pointer: `/${name}`,
          type: "float",
        }),
      ),
    ];
    const message: icd.Message = {
      key: uuid.create(),
      name: names[0],
      payload: {
        format: "json",
        match: { type: "present", pointer: `/${names[0]}` },
        fields,
      },
    };
    source.messages.push({
      message: message.key,
      index: indexField?.key,
      bindings: fields.map(({ key, name }) => ({
        field: key,
        channel: channels[name] ?? 0,
        dataType: "",
      })),
    });
    return { source, read, message, fields };
  });
};

/** Plans one message with what the Core holds, as a deploy does. */
const planMessage = async (
  ...args: Parameters<typeof ICD.planMessage> extends [unknown, ...infer A] ? A : never
): Promise<ICD.MessagePlan> =>
  await Task.planChannels(client, (planner) => ICD.planMessage(planner, ...args));

/**
 * Plans and then applies every message of bound with one planner, as a deploy does.
 *
 * @returns the names of the channels of each message, in binding order.
 */
const deployKinds = async (
  prefix: string,
  bound: ReturnType<typeof createKinds>,
): Promise<string[][]> => {
  const plans = await Task.planChannels(client, (planner) => {
    const written = new Set(
      bound
        .flatMap(({ source, read }) => source.messages[read].bindings)
        .map(({ channel }) => channel)
        .filter((key) => key !== 0),
    );
    return bound.map((b) =>
      ICD.planMessage(planner, b, prefix, undefined, written, bound),
    );
  });
  for (const [i, b] of bound.entries()) await ICD.applyMessage(client, b, plans[i]);
  return await Promise.all(
    bound.map(
      async ({ source, read }) =>
        await Promise.all(
          source.messages[read].bindings.map(
            async ({ channel }) => (await client.channels.retrieve(channel)).name,
          ),
        ),
    ),
  );
};

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
    const plan = await planMessage(bound, prefix);
    expect(plan.fields[0].replaces).toBe(0);
    await ICD.applyMessage(client, bound, plan);
    const key = bound.source.messages[0].bindings[0].channel;
    expect(key).not.toBe(old.key);
    expect((await client.channels.retrieve(key)).name).toBe(`${prefix}_temp`);
  });

  it("should name a new index after its index field", async () => {
    const prefix = uniqueName("src");
    const bound = createBound({ temp: 0, ts: 0 }, "ts");
    const plan = await planMessage(bound, prefix);
    await ICD.applyMessage(client, bound, plan);
    const [temp, ts] = bound.source.messages[0].bindings.map(({ channel }) => channel);
    const index = await client.channels.retrieve(ts);
    expect(index.name).toBe(`${prefix}_ts`);
    expect((await client.channels.retrieve(temp)).index).toBe(index.key);
  });

  it("should plan a float64 channel for a scaled integer field", async () => {
    const bound = createBound({ count: 0 }, undefined, { type: "integer", scale: 0.5 });
    const plan = await planMessage(bound, uniqueName("src"));
    expect(plan.fields[0].dataType).toBe("float64");
  });

  it("should replace the integer channel of a field that gains a scale", async () => {
    const prefix = uniqueName("src");
    const index = await createIndex(`${prefix}_time`);
    const old = await client.channels.create({
      name: `${prefix}_old`,
      dataType: "int64",
      index: index.key,
    });
    const bound = createBound({ count: old.key }, undefined, {
      type: "integer",
      scale: 0.5,
    });
    const plan = await planMessage(bound, prefix);
    expect(plan.fields[0].replaces).toBe(old.key);
    await ICD.applyMessage(client, bound, plan);
    const key = bound.source.messages[0].bindings[0].channel;
    expect(key).not.toBe(old.key);
    expect((await client.channels.retrieve(key)).dataType.toString()).toBe("float64");
  });

  it("should not reuse a stored integer channel for a scaled field", async () => {
    const prefix = uniqueName("src");
    const index = await createIndex(`${prefix}_time`);
    const old = await client.channels.create({
      name: `${prefix}_old`,
      dataType: "int64",
      index: index.key,
    });
    const bound = createBound({ count: 0 }, undefined, { type: "integer", offset: 4 });
    const props = { index: index.key, channels: { "/count": old.key } };
    const plan = await planMessage(bound, prefix, props);
    await ICD.applyMessage(client, bound, plan);
    const key = bound.source.messages[0].bindings[0].channel;
    expect(key).not.toBe(0);
    expect(key).not.toBe(old.key);
    expect((await client.channels.retrieve(key)).dataType.toString()).toBe("float64");
  });

  it("should store a scaled field whose stored integer channel is virtual", async () => {
    const prefix = uniqueName("src");
    const old = await client.channels.create({
      name: `${prefix}_old`,
      dataType: "int64",
      virtual: true,
    });
    const bound = createBound({ count: 0 }, undefined, { type: "integer", scale: 0.5 });
    const props = { index: 0, channels: { "/count": old.key } };
    const plan = await planMessage(bound, prefix, props);
    await ICD.applyMessage(client, bound, plan);
    const ch = await client.channels.retrieve(
      bound.source.messages[0].bindings[0].channel,
    );
    expect(ch.key).not.toBe(old.key);
    expect(ch.virtual).toBe(false);
    expect(ch.dataType.toString()).toBe("float64");
  });

  it("should drop an integer data type that the binding of a scaled field names", async () => {
    const bound = createBound({ count: 0 }, undefined, { type: "integer", scale: 0.5 });
    bound.source.messages[0].bindings[0].dataType = "int64";
    const plan = await planMessage(bound, uniqueName("src"));
    expect(plan.fields[0].dataType).toBe("float64");
    expect(plan.fields[0].binding.dataType).toBe("");
  });

  it("should plan a new channel when the stored channel was deleted", async () => {
    const prefix = uniqueName("src");
    const index = await createIndex(`${prefix}_time`);
    const old = await client.channels.create({
      name: `${prefix}_old`,
      dataType: "float64",
      index: index.key,
    });
    await client.channels.delete(old.key);
    const bound = createBound({ temp: 0 });
    const props = { index: index.key, channels: { "/temp": old.key } };
    const plan = await planMessage(bound, prefix, props);
    await ICD.applyMessage(client, bound, plan);
    const key = bound.source.messages[0].bindings[0].channel;
    expect(key).not.toBe(0);
    expect(key).not.toBe(old.key);
    expect((await client.channels.retrieve(key)).name).toBe(`${prefix}_temp`);
  });

  it("should keep its index and take back a stored channel that another message borrows", async () => {
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
    const plan = await planMessage(bound, prefix, props, new Set([shared.key]));
    await ICD.applyMessage(client, bound, plan);
    const [time, ...data] = await Promise.all(
      bound.source.messages[0].bindings.map(
        async ({ channel }) => await client.channels.retrieve(channel),
      ),
    );
    expect(time.key).toBe(index.key);
    expect(data.map(({ index }) => index)).toEqual([time.key, time.key, time.key]);
    expect(data.map(({ key }) => key)).toEqual([shared.key, stored.key, onIndex.key]);
  });

  it("should leave an index that another message writes when its bindings write nothing on it", async () => {
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
    const bound = createBound({ time: 0, temp: 0, pressure: 0 }, "time");
    const props = {
      index: index.key,
      channels: { "/temp": shared.key, "/pressure": stored.key },
    };
    const plan = await planMessage(bound, prefix, props, new Set([shared.key]));
    await ICD.applyMessage(client, bound, plan);
    const [time, ...data] = await Promise.all(
      bound.source.messages[0].bindings.map(
        async ({ channel }) => await client.channels.retrieve(channel),
      ),
    );
    expect(time.key).not.toBe(index.key);
    expect(data.map(({ index }) => index)).toEqual([time.key, time.key]);
    expect(data.map(({ name }) => name)).toEqual([
      `${prefix}_temp_2`,
      `${prefix}_pressure`,
    ]);
  });

  it("should leave a stored virtual channel that another message writes", async () => {
    const prefix = uniqueName("src");
    const createVirtual = async (name: string) =>
      await client.channels.create({
        name: `${prefix}_${name}`,
        dataType: "string",
        virtual: true,
      });
    const own = await createVirtual("mode");
    const shared = await createVirtual("state");
    const bound = createBound({ mode: own.key, state: 0 }, undefined, {
      type: "string",
    });
    const plan = await planMessage(
      bound,
      prefix,
      { index: 0, channels: { "/state": shared.key } },
      new Set([shared.key]),
    );
    await ICD.applyMessage(client, bound, plan);
    const [mode, state] = bound.source.messages[0].bindings.map(
      ({ channel }) => channel,
    );
    expect(mode).toBe(own.key);
    expect(state).not.toBe(0);
    expect(state).not.toBe(shared.key);
  });

  it("should leave a stored index that another message writes", async () => {
    const prefix = uniqueName("src");
    const index = await createIndex(`${prefix}_time`);
    const temp = await client.channels.create({
      name: `${prefix}_temp`,
      dataType: "float64",
      index: index.key,
    });
    const bound = createBound({ flow: 0 });
    const plan = await planMessage(
      bound,
      prefix,
      { index: index.key, channels: {} },
      new Set([temp.key]),
    );
    await ICD.applyMessage(client, bound, plan);
    const [{ channel }] = bound.source.messages[0].bindings;
    const flow = await client.channels.retrieve(channel);
    expect(flow.index).not.toBe(0);
    expect(flow.index).not.toBe(index.key);
  });

  describe("a source with a message for each kind", () => {
    it("should name both new indexes of the same name after their kinds", async () => {
      const prefix = uniqueName("src");
      const bound = createKinds([
        { index: "ts", fields: ["rpm"] },
        { index: "ts", fields: ["code"] },
      ]);
      expect(await deployKinds(prefix, bound)).toEqual([
        [`${prefix}_rpm_ts`, `${prefix}_rpm`],
        [`${prefix}_code_ts`, `${prefix}_code`],
      ]);
    });

    it("should name only the new index after its kind when the other exists", async () => {
      const prefix = uniqueName("src");
      const index = await createIndex(`${prefix}_ts`);
      const rpm = await client.channels.create({
        name: `${prefix}_rpm`,
        dataType: "float64",
        index: index.key,
      });
      const bound = createKinds([
        { index: "ts", fields: ["rpm"], channels: { ts: index.key, rpm: rpm.key } },
        { index: "ts", fields: ["code"] },
      ]);
      expect(await deployKinds(prefix, bound)).toEqual([
        [`${prefix}_ts`, `${prefix}_rpm`],
        [`${prefix}_code_ts`, `${prefix}_code`],
      ]);
    });

    it("should name both new indexes of arrival times after their kinds", async () => {
      const prefix = uniqueName("src");
      const bound = createKinds([{ fields: ["rpm"] }, { fields: ["code"] }]);
      await deployKinds(prefix, bound);
      const indexes = await Promise.all(
        bound.map(async ({ source, read }) => {
          const [{ channel: key }] = source.messages[read].bindings;
          const { index } = await client.channels.retrieve(key);
          return (await client.channels.retrieve(index)).name;
        }),
      );
      expect(indexes).toEqual([`${prefix}_rpm_time`, `${prefix}_code_time`]);
    });

    it("should name both new value channels of the same name after their kinds", async () => {
      const prefix = uniqueName("src");
      const bound = createKinds([
        { index: "ts", fields: ["rpm", "status"] },
        { index: "time", fields: ["code", "status"] },
      ]);
      expect(await deployKinds(prefix, bound)).toEqual([
        [`${prefix}_ts`, `${prefix}_rpm`, `${prefix}_rpm_status`],
        [`${prefix}_time`, `${prefix}_code`, `${prefix}_code_status`],
      ]);
    });
  });
});
