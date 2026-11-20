// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ranger, Synnax } from "@synnaxlabs/client";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { onTestFinished } from "vitest";

import { createKeyEndingIn } from "@/testutil";

/**
 * Creates a real range on the connected Core with a unique name and a valid, non-zero
 * time range, so range specs share one place for building fixtures. The overrides
 * replace those defaults. The range is deleted once the test finishes, so call it only
 * inside a test.
 */
export const createTestRange = async (
  client: Synnax,
  overrides: Partial<ranger.New> = {},
): Promise<ranger.Range> => {
  // Each start or end after now makes the stage readers of parallel specs, such as the
  // range toolbar, read again, so the range ends in the past.
  const end = TimeStamp.now().sub(TimeSpan.MINUTE);
  const rng = await client.ranges.create({
    name: uniqueName("range"),
    timeRange: new TimeRange(end.sub(TimeSpan.seconds(10)), end),
    ...overrides,
  });
  onTestFinished(async () => await client.ranges.delete(rng.key));
  return rng;
};

/**
 * Creates a unique range name for a range that the spec creates through the Console,
 * and deletes the ranges with that name once the test finishes.
 */
export const createTestRangeName = (client: Synnax, prefix: string): string => {
  const name = uniqueName(prefix);
  onTestFinished(async () => {
    const ranges = await client.ranges.retrieve([name]);
    await client.ranges.delete(ranges.map((r) => r.key));
  });
  return name;
};

/** Creates a range with no start. It sorts above every other range. */
export const createPlannedTestRange = async (
  client: Synnax,
  name: string,
): Promise<ranger.Range> =>
  await createTestRange(client, {
    name,
    timeRange: new TimeRange(TimeStamp.MAX, TimeStamp.MAX),
  });

/** Matches the default page size of Flux.useList, which the range select uses. */
const PAGE_SIZE = 50;

/**
 * Creates a range that no list shows on its first page, plus enough ranges ahead of it
 * to fill one. Gorp orders ranges by the trailing bytes of the key, so a key of all
 * ones sorts last and one of all zeros sorts first. Only a search reaches the returned
 * range, so a spec must assert it is absent before searching: that way a change to the
 * ordering fails the spec instead of quietly restoring it to the first page. All of the
 * ranges are deleted once the test finishes.
 */
export const createOffPageTestRange = async (client: Synnax): Promise<ranger.Range> => {
  const end = TimeStamp.now().sub(TimeSpan.MINUTE);
  const timeRange = new TimeRange(end.sub(TimeSpan.seconds(10)), end);
  const ahead = await client.ranges.create(
    Array.from({ length: PAGE_SIZE }, () => ({
      key: createKeyEndingIn("0"),
      name: uniqueName("ahead"),
      timeRange,
    })),
  );
  onTestFinished(async () => await client.ranges.delete(ahead.map((r) => r.key)));
  return await createTestRange(client, { key: createKeyEndingIn("f"), timeRange });
};
