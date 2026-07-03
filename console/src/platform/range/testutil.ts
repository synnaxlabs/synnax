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

/** Generates a cluster-safe unique range name: letters, digits, and underscores. */
export const uniqueRangeName = (prefix = "range"): string =>
  `${prefix}_${id.create().replace(/-/g, "_")}`;

/**
 * Creates a real range on the connected cluster with a unique name and a valid,
 * non-zero time range, so range specs share one place for building fixtures.
 */
export const createTestRange = async (client: Synnax): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueRangeName(),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(10))),
  });
};
