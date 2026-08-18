// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";

/** Generates a cluster-safe unique channel name: letters, digits, and underscores. */
export const uniqueChannelName = (prefix = "ch"): string =>
  `${prefix}_${id.create().replace(/-/g, "_")}`;

/** Creates a real timestamp index channel on the connected cluster. */
export const createTestIndexChannel = async (
  client: Synnax,
): Promise<channel.Channel> =>
  await client.channels.create({
    name: uniqueChannelName("idx"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
    leaseholder: 1,
  });

export interface TestIndexedPair {
  index: channel.Channel;
  data: channel.Channel;
}

/** Creates an index channel and a float64 data channel indexed by it. */
export const createTestIndexedPair = async (
  client: Synnax,
): Promise<TestIndexedPair> => {
  const index = await createTestIndexChannel(client);
  const data = await client.channels.create({
    name: uniqueChannelName("data"),
    dataType: DataType.FLOAT64,
    index: index.key,
    leaseholder: 1,
  });
  return { index, data };
};
