// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, TimeRange, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/index";
import { ranger } from "@/ranger";
import { createTestClient, spyOnSend, waitForStreamLive } from "@/testutil";
import { vectors } from "@/testutil/vectors";

const client = createTestClient();

describe("Where", () => {
  vectors.itFiltersLive(vectors.read("ranges", ranger.payloadZ, ranger.whereZ), () => {
    const local = createTestClient();
    let labels = new Map<string, string>();
    const keys = new Map<string, string>();
    return {
      client: local,
      target: "/range/retrieve",
      prepare: async (records) => {
        labels = await vectors.createLabels(
          local,
          records.flatMap(([, r]) => r.labels),
        );
        records.forEach(([, r]) => keys.set(r.key, uuid.create()));
        return {
          keys: new Map(records.map(([alias, r]) => [alias, keys.get(r.key) ?? r.key])),
          values: new Map<unknown, unknown>([...labels, ...keys]),
        };
      },
      write: async (records) => {
        await local.ranges.create(
          records.map(([, r]) => ({
            ...r,
            key: keys.get(r.key),
            labels: r.labels.map((l) => labels.get(l) ?? l),
          })),
        );
      },
      watch: (where) => local.ranges.watch({ where }),
    };
  });
});

describe("retrieve input", () => {
  const sentWhere = async (retrieve: () => Promise<unknown>): Promise<unknown> => {
    const send = spyOnSend(client);
    try {
      // A single retrieve fails when nothing matches; only the request matters here.
      const outcome = await retrieve().catch((e: unknown) => e);
      if (outcome instanceof Error) expect(NotFoundError.matches(outcome)).toBe(true);
      const [, req] =
        send.mock.calls.find(([target]) => target === "/range/retrieve") ?? [];
      return req;
    } finally {
      send.mockRestore();
    }
  };
  const labelKey = uuid.create();

  it("should send a value as eq", async () => {
    const name = `where-${uuid.create()}`;
    expect(
      await sentWhere(async () => await client.ranges.retrieve({ name })),
    ).toMatchObject({ where: { name: { eq: name } } });
  });

  it("should send a list as in", async () => {
    expect(
      await sentWhere(async () => await client.ranges.retrieve({ name: ["a", "b"] })),
    ).toMatchObject({
      where: { name: { in: ["a", "b"] } },
    });
  });

  it("should keep an operator", async () => {
    const name = { startsWith: "Hot", caseInsensitive: true };
    expect(
      await sentWhere(async () => await client.ranges.retrieve({ name })),
    ).toMatchObject({ where: { name } });
  });

  it("should send a value on a list line as a list of one", async () => {
    expect(
      await sentWhere(async () => await client.ranges.retrieve({ labels: labelKey })),
    ).toMatchObject({
      where: { labels: [labelKey] },
    });
  });

  it("should send the children of or", async () => {
    expect(
      await sentWhere(
        async () =>
          await client.ranges.retrieve({
            or: [{ name: { startsWith: "Hot" } }, { labels: labelKey }],
          }),
      ),
    ).toMatchObject({
      where: { or: [{ name: { startsWith: "Hot" } }, { labels: [labelKey] }] },
    });
  });

  it("should split the options from the fields", async () => {
    expect(
      await sentWhere(
        async () =>
          await client.ranges.retrieve({ name: ["a", "b"], limit: 2, searchTerm: "b" }),
      ),
    ).toMatchObject({ where: { name: { in: ["a", "b"] } }, limit: 2, searchTerm: "b" });
  });

  it("should leave where unset when only options are set", async () => {
    expect(
      await sentWhere(async () => await client.ranges.retrieve({ limit: 5 })),
    ).toMatchObject({ where: undefined });
  });

  it("should reject a field set to undefined", async () => {
    await expect(client.ranges.retrieve({ name: undefined })).rejects.toMatchObject({
      issues: [{ code: "invalid_union" }],
    });
  });

  it("should reject a field the filter does not have", async () => {
    // @ts-expect-error - the filter has no color field.
    await expect(client.ranges.retrieve({ color: "red" })).rejects.toMatchObject({
      issues: [{ code: "unrecognized_keys", keys: ["color"] }],
    });
  });
});

describe("live list order", () => {
  const [low, middle, high] = [uuid.create(), uuid.create(), uuid.create()].sort(
    compare.uuids,
  );

  const orderOf = async (
    order: ranger.Order | undefined,
    records: Array<{ key: string; name: string; start: number; end?: number }>,
  ): Promise<string[]> => {
    const local = createTestClient();
    await local.connect();
    await waitForStreamLive(local.connection);
    const keys = records.map(({ key }) => key);
    const list = local.ranges.watch({ where: { key: { in: keys } }, order });
    const close = list.onChange(() => {});
    try {
      await list.retrieve();
      await local.ranges.create(
        records.map(({ key, name, start, end }) => ({
          key,
          name,
          timeRange: new TimeRange(
            TimeStamp.seconds(start),
            TimeStamp.seconds(end ?? start + 1),
          ),
        })),
      );
      const held = () => list.getCached()?.records.map(({ key }) => key) ?? [];
      await expect.poll(() => held().length).toBe(records.length);
      return held();
    } finally {
      close();
    }
  };

  it("should order names by lower case, then by key", async () => {
    const [a, b, c] = [uuid.create(), uuid.create(), uuid.create()].sort(compare.uuids);
    expect(
      await orderOf({ field: "name", descending: false }, [
        { key: c, name: "alpha", start: 1 },
        { key: b, name: "Beta", start: 1 },
        { key: a, name: "ALPHA", start: 1 },
      ]),
    ).toEqual([a, c, b]);
  });

  it("should reverse the order and the key tie when descending", async () => {
    expect(
      await orderOf({ field: "time_range_start", descending: true }, [
        { key: low, name: "a", start: 1 },
        { key: middle, name: "a", start: 1 },
        { key: high, name: "a", start: 2 },
      ]),
    ).toEqual([high, middle, low]);
  });

  it("should order by end time, then by key", async () => {
    const [a, b, c] = [uuid.create(), uuid.create(), uuid.create()].sort(compare.uuids);
    expect(
      await orderOf({ field: "time_range_end", descending: false }, [
        { key: a, name: "a", start: 1, end: 9 },
        { key: c, name: "a", start: 2, end: 5 },
        { key: b, name: "a", start: 3, end: 5 },
      ]),
    ).toEqual([b, c, a]);
  });

  it("should order by key alone with no order", async () => {
    const [a, b] = [uuid.create(), uuid.create()].sort(compare.uuids);
    expect(
      await orderOf(undefined, [
        { key: b, name: "a", start: 1 },
        { key: a, name: "b", start: 2 },
      ]),
    ).toEqual([a, b]);
  });
});
