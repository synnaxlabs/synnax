// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This example demonstrates how to write data to an index channel and its corresponding
// data channel in Synnax in a streaming fashion. Streaming data is ideal for live
// applications (such as data acquisition from a sensor) or for very large datasets that
// cannot be written all at once.

import { Rate, Synnax, TimeStamp } from "@synnaxlabs/client";

// Connect to a locally running, insecure Synnax cluster. If your connection parameters
// are different, enter them here.
const client = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
});

// Create an index channel that will be used to store our timestamps.
const timeChannel = await client.channels.create(
  {
    name: "stream_write_example_time",
    isIndex: true,
    dataType: "timestamp",
  },
  { retrieveIfNameExists: true },
);

// Create a data channel that will be used to store our fake sensor data.
const dataChannel1 = await client.channels.create(
  {
    name: "stream_write_example_data_1",
    dataType: "float32",
    index: timeChannel.key,
  },
  { retrieveIfNameExists: true },
);

const dataChannel2 = await client.channels.create(
  {
    name: "stream_write_example_data_2",
    dataType: "int32",
    index: timeChannel.key,
  },
  { retrieveIfNameExists: true },
);

// We'll start our write at the current time. This timestamps should be the same as or
// just before the first timestamp we write.
const start = TimeStamp.now();

// Set a rough rate of 25 Hz. This won't be exact because we're sleeping for a fixed
// amount of time, but it's close enough for demonstration purposes.
const roughRate = Rate.hz(25);

const writer = await client.openWriter({
  start,
  channels: [timeChannel.key, dataChannel1.key, dataChannel2.key],
});

try {
  let i = 0;
  while (true) {
    await new Promise((resolve) => setTimeout(resolve, roughRate.period.milliseconds));
    i++;
    const timestamp = TimeStamp.now();
    const data1 = Math.sin(i / 10);
    const data2 = i % 2;
    await writer.write({
      [timeChannel.key]: timestamp,
      [dataChannel1.key]: data1,
      [dataChannel2.key]: data2,
    });
    if (i % 60 == 0) console.log(`Writing sample ${i} at ${timestamp.toISOString()}`);
  }
} finally {
  // Close the writer and the client when you are done
  await writer.close();
  client.close();
}
