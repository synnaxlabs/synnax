// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware } from "@synnaxlabs/freighter";
import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import type Synnax from "@/client";
import { framer } from "@/framer";
import {
  createTestClient,
  createTestClientWithPolicy,
  waitForStreamLive,
} from "@/testutil";

const admin = createTestClient();

const READER_POLICY = {
  name: "reader",
  objects: [channel.resourceID(0), framer.TYPE_RESOURCE_ID],
  actions: ["retrieve" as const],
};

const SUBJECTS: Record<string, (middleware?: Middleware[]) => Promise<Synnax>> = {
  "an admin that never retrieved the calculation": async (middleware) =>
    createTestClient({ middleware }),
  "a reader that cannot read the calculation": async (middleware) =>
    await createTestClientWithPolicy(
      admin,
      { ...READER_POLICY, name: id.create() },
      undefined,
      { middleware },
    ),
};

const WHERES: Record<string, (prefix: string) => channel.Where> = {
  "not calculated": (prefix) => ({ calculated: false, name: { startsWith: prefix } }),
  "not of calculated": (prefix) => ({
    not: { calculated: true },
    name: { startsWith: prefix },
  }),
  "and of not calculated": (prefix) => ({
    and: [{ not: { calculated: true } }, { name: { startsWith: prefix } }],
  }),
};

const RACE_WHERES: Record<string, (prefix: string) => channel.Where> = {
  calculated: (prefix) => ({ calculated: true, name: { startsWith: prefix } }),
  "not virtual": (prefix) => ({ virtual: false, name: { startsWith: prefix } }),
};

describe("live calculated lists, breaker", () => {
  Object.entries(SUBJECTS).forEach(([subject, createSubject]) => {
    Object.entries(WHERES).forEach(([line, whereOf]) =>
      it(`should drop and admit an output for ${subject}, ${line}`, async () => {
        const prefix = `calc_brk_${id.create()}`;
        const output = await admin.channels.create({
          name: `${prefix}_output`,
          dataType: DataType.INT64,
          virtual: true,
        });
        const client = await createSubject();
        await client.connect();
        await waitForStreamLive(client.connection);
        const list = client.channels.watch({ where: whereOf(prefix) });
        const off = list.onChange(() => {});
        const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
        try {
          await list.retrieve();
          expect(keys()).toEqual([output.key]);
          const calc = await admin.calculations.create({
            output: output.key,
            expression: "return 2",
          });
          await expect.poll(() => keys()).toEqual([]);
          await admin.calculations.delete(calc.key);
          await expect.poll(() => keys()).toEqual([output.key]);
        } finally {
          off();
        }
      }),
    );

    it(`should drop an output whose calculation predates the client for ${subject}`, async () => {
      const prefix = `calc_brk_${id.create()}`;
      const calc = await admin.calculations.create({
        name: `${prefix}_output`,
        expression: "return 2",
      });
      const client = await createSubject();
      await client.connect();
      await waitForStreamLive(client.connection);
      const list = client.channels.watch({
        where: { calculated: true, name: { startsWith: prefix } },
      });
      const off = list.onChange(() => {});
      const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
      try {
        await list.retrieve();
        expect(keys()).toEqual([calc.output]);
        await admin.calculations.delete(calc.key);
        await expect.poll(() => keys()).toEqual([]);
      } finally {
        off();
      }
    });

    it(`should admit a new output that a create makes for ${subject}`, async () => {
      const prefix = `calc_brk_${id.create()}`;
      const client = await createSubject();
      await client.connect();
      await waitForStreamLive(client.connection);
      const list = client.channels.watch({
        where: { calculated: true, name: { startsWith: prefix } },
      });
      const off = list.onChange(() => {});
      const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
      try {
        await list.retrieve();
        expect(keys()).toEqual([]);
        const calc = await admin.calculations.create({
          name: `${prefix}_output`,
          expression: "return 2",
        });
        await expect.poll(() => keys()).toEqual([calc.output]);
      } finally {
        off();
      }
    });

    Object.entries(RACE_WHERES).forEach(([line, whereOf]) =>
      it(`should admit an output whose calculation starts during the first fetch for ${subject}, ${line}`, async () => {
        const prefix = `calc_brk_${id.create()}`;
        // The type of the expression, so the create changes no field of the channel.
        const output = await admin.channels.create({
          name: `${prefix}_output`,
          dataType: DataType.INT64,
          virtual: true,
        });
        // Holds the answer of the list's first fetch until the client knows of the
        // calculation, as a slow network would.
        let hold: Promise<void> | null = null;
        let release = (): void => {};
        let answered = (): void => {};
        const arrived = new Promise<void>((resolve) => {
          answered = resolve;
        });
        const gate: Middleware = async (ctx, next) => {
          const out = await next(ctx);
          if (hold != null && ctx.target.endsWith("/channel/retrieve")) {
            const held = hold;
            hold = null;
            answered();
            await held;
          }
          return out;
        };
        const client = await createSubject([gate]);
        await client.connect();
        await waitForStreamLive(client.connection);
        // A first list starts the cache's stream, so its epoch refetch lands before the
        // list under test exists.
        const warm = client.channels.watch({
          where: { name: { startsWith: id.create() } },
        });
        const offWarm = warm.onChange(() => {});
        await warm.retrieve();
        await new Promise((resolve) => setTimeout(resolve, 1000));
        const where = whereOf(prefix);
        const list = client.channels.watch({ where });
        const off = list.onChange(() => {});
        const keys = () => (list.getCached()?.records ?? []).map(({ key }) => key);
        try {
          hold = new Promise<void>((resolve) => {
            release = resolve;
          });
          const first = list.retrieve();
          await arrived;
          await admin.calculations.create({
            output: output.key,
            expression: "return 2",
          });
          await expect
            .poll(() => client.calculations.outputs.get(output.key))
            .toBeDefined();
          release();
          await first;
          const fresh = await admin.channels.retrieve({ where: { ...where } });
          expect(fresh.map(({ key }) => key)).toEqual([output.key]);
          await expect.poll(() => keys(), { timeout: 5000 }).toEqual([output.key]);
        } finally {
          off();
          offWarm();
        }
      }),
    );
  });

  it("should read the line as a top-level field", async () => {
    const prefix = `calc_brk_${id.create()}`;
    const plain = await admin.channels.create({
      name: `${prefix}_plain`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const calc = await admin.calculations.create({
      name: `${prefix}_output`,
      expression: "return 2",
    });
    const client = createTestClient();
    const calculated = await client.channels.retrieve({
      calculated: true,
      name: { startsWith: prefix },
    });
    expect(calculated.map(({ key }) => key)).toEqual([calc.output]);
    const rest = await client.channels.retrieve({
      calculated: false,
      key: [plain.key, calc.output],
    });
    expect(rest.map(({ key }) => key)).toEqual([plain.key]);
    expect(
      (await client.channels.retrieve({ key: calc.output, calculated: true })).key,
    ).toEqual(calc.output);
    await expect(
      client.channels.retrieve({ key: calc.output, calculated: false }),
    ).rejects.toThrow();
  });
});
