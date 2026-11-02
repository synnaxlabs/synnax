// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { test } from "vitest";

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

// Opens a latest subscription to each channel, as a page of values does, and resolves
// once every one loads.
const loadLatest = async (keys: channel.Key[]): Promise<void> => {
  const subs = keys.map((key) =>
    feed.subscribe({ keys: [key], limit: { variant: "latest", count: 1 } }),
  );
  try {
    await Promise.all(
      subs.map(
        async (sub, i) =>
          await new Promise<void>((resolve, reject) => {
            const check = (): void => {
              const { status } = sub.value(keys[i]);
              if (status.variant === "success") resolve();
              else if (status.variant === "error") reject(new Error(status.message));
            };
            sub.onChange(check);
            check();
          }),
      ),
    );
  } finally {
    subs.forEach((sub) => sub.close());
  }
};

const suite = (title: string, keysOf: Map<number, channel.Key[]>): void => {
  // A subscribed run waits for the stream to ack its channels, so a suite outlasts the
  // default timeout.
  test(title, { timeout: 0 }, async ({ bench }) => {
    await bench.compare(
      ...SIZES.flatMap((n) => {
        const keys = keysOf.get(n) ?? [];
        return [
          bench(`subscribed ${n}`, async () => await loadLatest(keys)),
          bench(`per key ${n}`, async () => {
            await Promise.all(keys.map(async (k) => await client.readLatest(k, 1)));
          }),
        ];
      }),
    );
  });
};

suite("latest stored", stored);
suite("latest calculated", calculated);
