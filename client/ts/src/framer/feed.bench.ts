// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { bench, describe } from "vitest";

import { type channel } from "@/channel";
import { createTestClient } from "@/testutil";

const SIZES = [1, 10, 80, 300];

const client = createTestClient();
// A zero window measures the Core, not the debounce.
const feed = client.openFeed({ batchDebounce: TimeSpan.ZERO });

const createStored = async (n: number): Promise<channel.Channel[]> => {
  const time = await client.channels.create({
    name: id.create(),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  const data = await client.channels.create(
    Array.from({ length: n }, () => ({
      name: id.create(),
      dataType: DataType.FLOAT32,
      index: time.key,
    })),
  );
  const now = TimeStamp.now();
  await client.write(now, {
    [time.key]: [now],
    ...Object.fromEntries(data.map((c) => [c.key, [1]])),
  });
  return data;
};

const createCalculated = async (n: number): Promise<channel.Channel[]> => {
  const [base] = await createStored(1);
  const created = await client.channels.create(
    Array.from({ length: n }, () => ({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
      expression: `return ${base.name}`,
    })),
  );
  return created.filter((c) => c.expression !== "");
};

const stored = new Map<number, channel.Key[]>();
const calculated = new Map<number, channel.Key[]>();
for (const n of SIZES) {
  stored.set(
    n,
    (await createStored(n)).map((c) => c.key),
  );
  calculated.set(
    n,
    (await createCalculated(n)).map((c) => c.key),
  );
}

const suite = (title: string, keysOf: Map<number, channel.Key[]>): void =>
  describe(title, () => {
    for (const n of SIZES) {
      const keys = keysOf.get(n) ?? [];
      bench(`batched ${n}`, async () => {
        await Promise.all(keys.map(async (k) => await feed.readLatest(k)));
      });
      bench(`per key ${n}`, async () => {
        await Promise.all(keys.map(async (k) => await client.readLatest(k, 1)));
      });
    }
  });

suite("readLatest stored", stored);
suite("readLatest calculated", calculated);
