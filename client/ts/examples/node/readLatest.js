// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * This example demonstrates how to read the latest N samples from Synnax channels. We'll:
 * 1. Create an index channel for timestamps and a data channel for values
 * 2. Write some sample data to these channels
 * 3. Read the latest N samples using an iterator
 * 4. Print the results
 *
 * This pattern is useful for real-time monitoring applications where you need to access
 * the most recent data points.
 */

import { DataType, Synnax, TimeSpan, TimeStamp } from "@synnaxlabs/client";

// Connect to a locally running, insecure Synnax cluster. If your connection parameters
// are different, enter them here. See https://docs.synnaxlabs.com/reference/client/quick-start.
const client = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
});

// Create an index channel that will be used to store our timestamps.
const timeChannel = await client.channels.create({
  name: "read_latest_time",
  isIndex: true,
  dataType: DataType.TIMESTAMP,
});

// Create a data channel that will be used to store our data.
const dataChannel = await client.channels.create({
  name: "read_latest_data",
  // We need to specify the index channel that will be used to store the timestamps for
  // this data channel.
  index: timeChannel.key,
  dataType: DataType.FLOAT32,
});

const SAMPLE_COUNT = 100;

// We'll start our write at the current time. This timestamp should be the same as or
// just before the first timestamp we write.
const start = TimeStamp.now();

// We'll end our write 100 seconds later
const end = start.add(TimeSpan.seconds(2));

// Generate linearly spaced int64 timestamps
const stamps = BigInt64Array.from({ length: SAMPLE_COUNT }, (_, i) =>
  start.add(TimeSpan.milliseconds((i * 2000) / SAMPLE_COUNT)).valueOf(),
);

// Generate a sine wave with some noise as our data
const data = Float32Array.from(
  { length: SAMPLE_COUNT },
  (_, i) => Math.sin(1 + (i * 9) / SAMPLE_COUNT) * 20 + (Math.random() - 0.5),
);

// Write the data to the Synnax cluster through the channels. Note that we need to write
// to the index channel first, otherwise our write will fail.
await timeChannel.write(start, stamps);
await dataChannel.write(start, data);

console.log(await client.readLatest(["read_latest_time", "read_latest_data"], 1));

client.close();
