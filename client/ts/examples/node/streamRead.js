// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";

// This example demonstrates how to stream live data from a channel in Synnax.
// Live-streaming is useful for real-time data processing and analysis, and is an
// integral part of Synnax's control sequence and data streaming capabilities.

// This example is meant to be used in conjunction with the stream_write.js example, and
// assumes that example is running in a separate terminal.

// Connect to a locally running, insecure Synnax cluster. If your connection parameters
// are different, enter them here.
const client = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
});

// We can just specify the names of the channels we'd like to stream from.
const read_from = [
  "stream_write_example_time",
  "stream_write_example_data_1",
  "stream_write_example_data_2",
];

const streamer = await client.openStreamer(read_from);

// It's very important that we close the streamer when we're done with it to release
// network connections and other resources, so we wrap the streaming loop in a
// try-finally block.
try {
  // Loop through the frames in the streamer. Each iteration will block until a new
  // frame is available, and then we'll just print out the last sample for each
  // channel in the frame.
  for await (const frame of streamer) console.log(frame.at(-1));
} finally {
  streamer.close();
  // Close the client when we're done with it.
  client.close();
}
