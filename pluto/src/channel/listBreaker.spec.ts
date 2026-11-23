// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  calculation,
  channel,
  DataType,
  framer,
  type Synnax,
  type SynnaxParams,
} from "@synnaxlabs/client";
import {
  createSeverableProxy,
  createTestClient,
  createTestClientWithPolicy,
  FAST_RETRY,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import type { Middleware } from "@synnaxlabs/freighter";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil";

const admin = createTestClient();

// The line of the line plot axis select and the CSV download.
const withData = (prefix: string): Channel.ListQuery => ({
  where: {
    or: [{ virtual: false }, { calculated: true }],
    name: { startsWith: prefix },
  },
});

// The line of the two Sift pickers.
const stored = (prefix: string): Channel.ListQuery => ({
  where: {
    virtual: false,
    calculated: false,
    isIndex: { eq: false },
    internal: { eq: false },
    name: { startsWith: prefix },
  },
});

const READER_POLICY = {
  objects: [channel.TYPE_RESOURCE_ID, framer.TYPE_RESOURCE_ID],
  actions: ["retrieve" as const],
};

const createSubject = async (params?: Partial<SynnaxParams>): Promise<Synnax> => {
  const client = createTestClient(params);
  await client.connect();
  await waitForStreamLive(client.connection);
  return client;
};

// A reader that can retrieve channels but not calculations.
const createReader = async (params?: Partial<SynnaxParams>): Promise<Synnax> => {
  const reader = await createTestClientWithPolicy(
    admin,
    { ...READER_POLICY, name: id.create() },
    undefined,
    params,
  );
  await reader.connect();
  await waitForStreamLive(reader.connection);
  return reader;
};

const renderList = async (client: Synnax, query: Channel.ListQuery) => {
  const wrapper = await createAsyncSynnaxWrapper({ client });
  const { result } = renderHook(() => Channel.useList({ query }), { wrapper });
  act(() => result.current.more());
  return result;
};

// Once armed, holds the answer to the next channel retrieve until release, after the
// Core answered it: a slow reply.
const createHold = () => {
  const released = Promise.withResolvers<void>();
  const answered = Promise.withResolvers<void>();
  let armed = false;
  const middleware: Middleware = async (ctx, next) => {
    const out = await next(ctx);
    if (armed && ctx.target.endsWith("/channel/retrieve")) {
      armed = false;
      answered.resolve();
      await released.promise;
    }
    return out;
  };
  onTestFinished(() => released.resolve());
  return {
    params: { middleware: [middleware] },
    arm: () => {
      armed = true;
    },
    answered: answered.promise,
    release: () => released.resolve(),
  };
};

// Resolves once the client's own calculated list shows keys, so the client has the
// change that the Core announced.
const waitForCalculated = async (
  client: Synnax,
  prefix: string,
  keys: channel.Key[],
): Promise<void> => {
  const list = client.channels.watch({
    where: { calculated: true, name: { startsWith: prefix } },
  });
  const off = list.onChange(() => {});
  onTestFinished(off);
  await list.retrieve();
  await expect
    .poll(() => (list.getCached()?.records ?? []).map(({ key }) => key))
    .toEqual(keys);
};

const createOutput = async (prefix: string) =>
  await admin.channels.create({
    name: `${prefix}_output`,
    dataType: DataType.FLOAT32,
    virtual: true,
  });

describe("channel lists with the calculated line", () => {
  it("should list a stored data channel and leave out virtual and calculated ones", async () => {
    const prefix = `stored_${id.create()}`;
    const idx = await admin.channels.create({
      name: `${prefix}_idx`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const data = await admin.channels.create({
      name: `${prefix}_data`,
      dataType: DataType.FLOAT32,
      index: idx.key,
    });
    await createOutput(prefix);
    await admin.calculations.create({
      name: `${prefix}_calc`,
      expression: `return ${data.name} * 2`,
    });
    const result = await renderList(await createSubject(), stored(prefix));
    await waitFor(() => expect(result.current.data).toEqual([data.key]));
  });

  it("should keep a calculated output out of a searched stored list", async () => {
    const prefix = `search_${id.create()}`;
    const idx = await admin.channels.create({
      name: `${prefix}_idx`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const data = await admin.channels.create({
      name: `${prefix}_data`,
      dataType: DataType.FLOAT32,
      index: idx.key,
    });
    const calc = await admin.calculations.create({
      name: `${prefix}_calc`,
      expression: `return ${data.name} * 2`,
    });
    const wrapper = await createAsyncSynnaxWrapper({ client: await createSubject() });
    const { result } = renderHook(
      () => ({
        stored: Channel.useList({ query: stored(prefix) }),
        withData: Channel.useList({ query: withData(prefix) }),
      }),
      { wrapper },
    );
    act(() => {
      result.current.stored.more();
      result.current.withData.more();
    });
    act(() => {
      result.current.stored.search(prefix);
      result.current.withData.search(prefix);
    });
    await waitFor(() =>
      expect([...result.current.withData.data].sort()).toEqual(
        [idx.key, data.key, calc.output].sort(),
      ),
    );
    await waitFor(() => expect(result.current.stored.data).toEqual([data.key]));
  });

  it("should drop an output from the axis list when its expression turns it to a string", async () => {
    const prefix = `type_${id.create()}`;
    const calc = await admin.calculations.create({
      name: `${prefix}_calc`,
      expression: "return 2.0",
    });
    const result = await renderList(await createSubject(), {
      where: {
        not: { dataType: { in: [DataType.STRING, DataType.JSON, DataType.UUID] } },
        or: [{ virtual: false }, { calculated: true }],
        name: { startsWith: prefix },
      },
    });
    await waitFor(() => expect(result.current.data).toEqual([calc.output]));
    await admin.calculations.dispatch(
      calc.key,
      calculation.set({ path: ["expression"], value: 'return "a"' }),
    );
    await expect
      .poll(async () =>
        (await admin.channels.retrieve(calc.output)).dataType.toString(),
      )
      .toBe(DataType.STRING.toString());
    await waitFor(() => expect(result.current.data).toEqual([]));
  });

  const SUBJECTS: Record<string, (p?: Partial<SynnaxParams>) => Promise<Synnax>> = {
    "an admin": createSubject,
    "a reader that cannot read the calculation": createReader,
  };

  Object.entries(SUBJECTS).forEach(([subject, create]) =>
    describe(`for ${subject}`, () => {
      it("should admit an output that a calculation starts to write", async () => {
        const prefix = `start_${id.create()}`;
        const output = await createOutput(prefix);
        const result = await renderList(await create(), withData(prefix));
        await waitFor(() => expect(result.current.variant).toBe("success"));
        expect(result.current.data).toEqual([]);
        await admin.calculations.create({ output: output.key, expression: "return 2" });
        await waitFor(() => expect(result.current.data).toEqual([output.key]));
      });

      it("should drop an output when its calculation is deleted", async () => {
        const prefix = `stop_${id.create()}`;
        const output = await createOutput(prefix);
        const calc = await admin.calculations.create({
          output: output.key,
          expression: "return 2",
        });
        const result = await renderList(await create(), withData(prefix));
        await waitFor(() => expect(result.current.data).toEqual([output.key]));
        await admin.calculations.delete(calc.key);
        await waitFor(() => expect(result.current.data).toEqual([]));
      });

      it("should admit an output whose calculation is created while the first answer is slow", async () => {
        const prefix = `slow_start_${id.create()}`;
        const output = await createOutput(prefix);
        const hold = createHold();
        const client = await create(hold.params);
        await waitForCalculated(client, prefix, []);
        hold.arm();
        const result = await renderList(client, withData(prefix));
        await hold.answered;
        await admin.calculations.create({ output: output.key, expression: "return 2" });
        await waitForCalculated(client, prefix, [output.key]);
        hold.release();
        await waitFor(() => expect(result.current.variant).toBe("success"));
        await waitFor(() => expect(result.current.data).toEqual([output.key]));
      });

      it("should drop an output whose calculation is deleted while the first answer is slow", async () => {
        const prefix = `slow_stop_${id.create()}`;
        const output = await createOutput(prefix);
        const calc = await admin.calculations.create({
          output: output.key,
          expression: "return 2",
        });
        const hold = createHold();
        const client = await create(hold.params);
        await waitForCalculated(client, prefix, [output.key]);
        hold.arm();
        const result = await renderList(client, withData(prefix));
        await hold.answered;
        await admin.calculations.delete(calc.key);
        await waitForCalculated(client, prefix, []);
        hold.release();
        await waitFor(() => expect(result.current.variant).toBe("success"));
        await waitFor(() => expect(result.current.data).toEqual([]));
      });

      it("should drop an output whose calculation is deleted while the stream is down", async () => {
        const prefix = `lost_${id.create()}`;
        const output = await createOutput(prefix);
        const calc = await admin.calculations.create({
          output: output.key,
          expression: "return 2",
        });
        const proxy = await createSeverableProxy();
        const client = await create({
          host: "localhost",
          port: proxy.port,
          retry: FAST_RETRY,
        });
        onTestFinished(async () => {
          await proxy.close();
          await client.close();
        });
        const result = await renderList(client, withData(prefix));
        await waitFor(() => expect(result.current.data).toEqual([output.key]));
        await proxy.sever();
        await admin.calculations.delete(calc.key);
        await proxy.restore();
        await waitForStreamLive(client.connection);
        await waitFor(() => expect(result.current.data).toEqual([]), {
          timeout: 15000,
        });
      }, 30000);
    }),
  );

  // virtual: false is the line plot line on the base, where an output counts as not
  // virtual.
  it("should drop an output from the base line for a reader when its calculation is deleted while the first answer is slow", async () => {
    const prefix = `base_slow_stop_${id.create()}`;
    const output = await createOutput(prefix);
    const calc = await admin.calculations.create({
      output: output.key,
      expression: "return 2",
    });
    const hold = createHold();
    const reader = await createReader(hold.params);
    await waitForCalculated(reader, prefix, [output.key]);
    hold.arm();
    const result = await renderList(reader, {
      where: { virtual: false, name: { startsWith: prefix } },
    });
    await hold.answered;
    await admin.calculations.delete(calc.key);
    await waitForCalculated(reader, prefix, []);
    hold.release();
    await waitFor(() => expect(result.current.variant).toBe("success"));
    await waitFor(() => expect(result.current.data).toEqual([]));
  });
});
