// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, type Synnax } from "@synnaxlabs/client";
import { id, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { createKeyEndingIn } from "@/testutil";

/** Generates a Core-safe unique range name: letters, digits, and underscores. */
export const uniqueRangeName = (prefix = "range"): string =>
  `${prefix}_${id.create().replace(/-/g, "_")}`;

/**
 * Creates a real range on the connected Core with a unique name and a valid, non-zero
 * time range, so range specs share one place for building fixtures.
 */
export const createTestRange = async (client: Synnax): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueRangeName(),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
  });
};

/** Creates a child of parent on the connected Core, named with the child prefix. */
export const createChildTestRange = async (
  client: Synnax,
  parent: ranger.Range,
): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueRangeName("child"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
    parent: { key: parent.key },
  });
};

/** Matches the default page size of List.usePager, which every range list uses. */
const PAGE_SIZE = 10;

/**
 * Creates a range that no list shows on its first page, plus enough ranges ahead of it
 * to fill one. Gorp orders ranges by the trailing bytes of the key, so a key of all
 * ones sorts last and one of all zeros sorts first. Only a search reaches the returned
 * range, so a spec must assert it is absent before searching: that way a change to the
 * ordering fails the spec instead of quietly restoring it to the first page.
 */
export const createOffPageTestRange = async (client: Synnax): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  const timeRange = new TimeRange(start, start.add(TimeSpan.seconds(10)));
  await client.ranges.create(
    Array.from({ length: PAGE_SIZE }, () => ({
      key: createKeyEndingIn("0"),
      name: uniqueRangeName("ahead"),
      timeRange,
    })),
  );
  return await client.ranges.create({
    key: createKeyEndingIn("f"),
    name: uniqueRangeName(),
    timeRange,
  });
};
