// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * This example shows how to create a named range in Synnax, which can be used to identify
 * and lookup specific periods of time in your data.
 *
 * We'll write  data to an index and data channel, and then create a range that spans the
 * entire time range of the data. Then, we'll show how to read the data back using the
 * range.
 */

import { DataType, Synnax, TimeSpan, TimeStamp } from "@synnaxlabs/client";

// Connect to a locally running, insecure Synnax cluster. If your connection parameters
// are different, enter them here. See https://docs.synnaxlabs.com/reference/client/quick-start.
const client = new Synnax({
});

// Define the data.
const start = TimeStamp.now();
const end = start.add(TimeSpan.seconds(10));

const timeData = BigInt64Array.from({ length: 1000 }, (_, i) =>
  start.add(TimeSpan.milliseconds(i * 10)).valueOf(),
);
const data = Float64Array.from({ length: 1000 }, (_, i) => Math.sin(i * 0.01));

// Create an index channel that will be used to store our timestamps.
const timeChannel = await client.channels.create({
  name: "create_range_time",
  dataType: DataType.TIMESTAMP,
  isIndex: true,
});

// Create a data channel that will be used to store our fake sensor data.
const dataChannel = await client.channels.create({
  name: "create_range_data",
  dataType: DataType.FLOAT64,
  index: timeChannel.key,
});

// Write the data to the Synnax cluster through the channels. Note that we need to write
// to the index channel first, otherwise our write will fail.
await timeChannel.write(start, timeData);
await dataChannel.write(start, data);

// Create a range that spans the start and end of the data.
const exampleRange = await client.ranges.create({
  name: "create_range_range",
  timeRange: { start, end },
});

// We can pull and plot the data from the range by just accessing the channel names as if
// they were attributes of the range itself.
const rangeTimeData = await exampleRange.read("create_range_time");
const rangeData = await exampleRange.read("create_range_data");

console.log("Time:", rangeTimeData.at(0), "to", rangeTimeData.at(-1));
console.log("Data:", rangeData.at(0), "to", rangeData.at(-1));

client.close();
