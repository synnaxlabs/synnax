// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Synnax, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

// This example demonstrates the basics of reading and writing data from an index and
// data channel in Synnax. We'll write a sine wave of data to a channel and then read it
// back.

// Connect to a locally running, insecure Synnax cluster. If your connection parameters
// are different, enter them here.
const client = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
});

// Create an index channel that will be used to store our timestamps.
const timeChannel = await client.channels.create(
  {
    name: "basic_read_write_time",
    isIndex: true,
    dataType: DataType.TIMESTAMP,
  },
  { retrieveIfNameExists: true },
);

// Create a channel that will be used to store our data.
const dataChannel = await client.channels.create(
  {
    name: "basic_read_write_data",
    isIndex: false,
    dataType: DataType.FLOAT32,
    // We need to specify the index channel that we want to use to store the timestamps
    // for this data channel.
    index: timeChannel.key,
  },
  { retrieveIfNameExists: true },
);

const N_SAMPLES = 5000;

// We'll start our write at the current time. This timestamp should be the same as or
// just before the first timestamp we write.
const start = TimeStamp.now();

// Generate a new timestamp every millisecond for N_SAMPLES.
const time = BigInt64Array.from({ length: N_SAMPLES }, (_, i) =>
  start.add(TimeSpan.milliseconds(i)).valueOf(),
);

// Generate a sine wave for N_SAMPLES.
const data = Float32Array.from({ length: N_SAMPLES }, (_, i) => Math.sin(i / 100));

// Write the data to the channel. Note that we need to write the timestamps first,
// otherwise writing the data will fail. Notice how we align the writes with the 'start'
// timestamp.
await timeChannel.write(start, time);
await dataChannel.write(start, data);

// Define the time range to read the data back from
const tr = new TimeRange(start, start.add(TimeSpan.milliseconds(N_SAMPLES)));

// Read the data back. The order doesn't matter here.
const readTime = await timeChannel.read(tr);
const readData = await dataChannel.read(tr);

// Print out some information.
console.log({
  firstTimestamp: readTime.at(0),
  firstData: readData.at(0),
  lastTimestamp: readTime.at(N_SAMPLES - 1),
  lastData: readData.at(N_SAMPLES - 1),
  returnLength: readData.length,
});

// Make sure to close the client when you're done.
client.close();
