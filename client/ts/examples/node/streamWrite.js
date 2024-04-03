// This example demonstrates how to write data to an index channel and its corresponding
// data channel in Synnax in a streaming fashion. Streaming data is ideal for live
// applications (such as data acquisition from a sensor) or for very large datasets that
// cannot be written all at once.

import { Synnax } from "@synnaxlabs/client";

// Connect to a locally running, insecure Synnax cluster. If your connection parameters
// are different, enter them here.
const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false
});

// Create an index channel that will be used to store our timestamps.
const timeChannel = await client.channels.create({
    name: "stream_write_example_time",
    isIndex: true,
    dataType: "timestamp"
});