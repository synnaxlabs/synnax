// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, sleep, TimeSpan } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { framer } from "@/framer";
import type { Synnax } from "@/index";
import {
  createFileTestClient,
  createTestClientWithPolicy,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const admin = createFileTestClient();

interface Fixture {
  prefix: string;
  /** Every channel under the prefix, and whether the old line called it virtual. */
  virtual: Map<channel.Key, boolean>;
}

const createFixture = async (prefix = `vw_${id.create()}`): Promise<Fixture> => {
  const virtual = new Map<channel.Key, boolean>();
  const input = await admin.channels.create({
    name: `${prefix}_in`,
    dataType: DataType.FLOAT32,
    virtual: true,
  });
  virtual.set(input.key, true);
  const index = await admin.channels.create({
    name: `${prefix}_time`,
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  virtual.set(index.key, false);
  const data = await admin.channels.create({
    name: `${prefix}_data`,
    dataType: DataType.FLOAT32,
    index: index.key,
  });
  virtual.set(data.key, false);
  const calc = await admin.calculations.create({
    name: `${prefix}_out`,
    expression: `return ${input.name} * 2`,
  });
  virtual.set(calc.output, false);
  const out = await admin.channels.retrieve(calc.output);
  // The calculation's own index is a plain virtual channel.
  virtual.set(out.index, true);
  const written = await admin.channels.create({
    name: `${prefix}_written`,
    dataType: DataType.FLOAT32,
    virtual: true,
  });
  await admin.calculations.create({
    output: written.key,
    expression: `return ${input.name} + 1`,
  });
  virtual.set(written.key, false);
  return { prefix, virtual };
};

const sorted = (keys: Iterable<channel.Key>) => [...keys].sort();

describe("virtual where breaker", () => {
  let fx: Fixture;
  let reader: Synnax;
  beforeAll(async () => {
    fx = await createFixture();
    reader = await createTestClientWithPolicy(admin, {
      name: id.create(),
      objects: [channel.resourceID(0)],
      actions: ["retrieve"],
    });
  }, 60_000);

  const expected = (keep: (virtual: boolean) => boolean, of: Fixture = fx) =>
    sorted([...of.virtual].filter(([, v]) => keep(v)).map(([k]) => k));

  const cases: Array<
    [string, (v: boolean) => object, (v: boolean, ch: boolean) => boolean]
  > = [
    ["alone", (v) => ({ virtual: v }), (v, ch) => ch === v],
    ["under and", (v) => ({ and: [{ virtual: v }] }), (v, ch) => ch === v],
    ["under or", (v) => ({ or: [{ virtual: v }] }), (v, ch) => ch === v],
    ["under not", (v) => ({ not: { virtual: v } }), (v, ch) => ch !== v],
    [
      "under not of not",
      (v) => ({ not: { not: { virtual: v } } }),
      (v, ch) => ch === v,
    ],
    [
      "or of both values",
      (v) => ({ or: [{ virtual: v }, { virtual: !v }] }),
      () => true,
    ],
    [
      "and of both values",
      (v) => ({ and: [{ virtual: v }, { virtual: !v }] }),
      () => false,
    ],
    [
      "beside a not of itself",
      (v) => ({ virtual: v, not: { virtual: v } }),
      () => false,
    ],
    [
      "under or beside a key",
      (v) => ({ or: [{ virtual: v }, { key: { in: [0] } }] }),
      (v, ch) => ch === v,
    ],
    [
      "beside a not of a data type",
      (v) => ({
        virtual: v,
        not: { dataType: { in: ["string", "json", "uuid"] } },
      }),
      (v, ch) => ch === v,
    ],
  ];

  for (const subject of ["admin", "reader"] as const)
    for (const value of [true, false])
      for (const [name, where, keep] of cases)
        it(`should keep the old line's set for ${subject}, virtual ${value}, ${name}`, async () => {
          const c = subject === "admin" ? admin : reader;
          const got = await c.channels.retrieve({
            where: channel.whereZ.parse({
              and: [where(value), { name: { startsWith: fx.prefix } }],
            }),
          });
          expect(sorted(got.map(({ key }) => key))).toEqual(
            expected((ch) => keep(value, ch)),
          );
        });

  // The lists open before the channels exist, so the client's own virtual line, not the
  // Core, decides each case as the channels and calculations arrive.
  for (const subject of ["admin", "reader"] as const)
    it(`should keep the old line's set in live lists for ${subject} as the channels arrive`, async () => {
      const c =
        subject === "admin"
          ? admin
          : await createTestClientWithPolicy(admin, {
              name: id.create(),
              objects: [channel.resourceID(0), framer.resourceID("")],
              actions: ["retrieve"],
            });
      await waitForStreamLive(c.connection);
      const prefix = `vw_${id.create()}`;
      const lists = [true, false].flatMap((value) =>
        cases.map(([name, where, keep]) => ({
          name: `virtual ${value}, ${name}`,
          keep: (ch: boolean) => keep(value, ch),
          list: c.channels.watch({
            where: channel.whereZ.parse({
              and: [where(value), { name: { startsWith: prefix } }],
            }),
          }),
        })),
      );
      const closers = lists.map(({ list }) => list.onChange(() => {}));
      try {
        await Promise.all(lists.map(async ({ list }) => await list.retrieve()));
        const send = spyOnSend(c);
        const live = await createFixture(prefix);
        const got = () =>
          lists.map(({ name, list }) => ({
            name,
            keys: sorted(list.getCached()?.records.map(({ key }) => key) ?? []),
          }));
        const want = lists.map(({ name, keep }) => ({
          name,
          keys: expected(keep, live),
        }));
        await expect.poll(got, { timeout: 10_000 }).toEqual(want);
        // Past the debounce of a refetch that a change could start.
        await sleep.sleep(TimeSpan.milliseconds(500));
        expect(got()).toEqual(want);
        const refetches = send.mock.calls.filter(
          ([target, req]) =>
            target === "/channel/retrieve" &&
            JSON.stringify((req as { where?: unknown }).where ?? null).includes(prefix),
        );
        expect(refetches).toHaveLength(0);
      } finally {
        closers.forEach((close) => close());
      }
    }, 60_000);
});
