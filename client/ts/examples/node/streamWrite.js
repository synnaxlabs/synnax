// This example demonstrates how to write data to an index channel and its corresponding
// data channel in Synnax in a streaming fashion. Streaming data is ideal for live
// applications (such as data acquisition from a sensor) or for very large datasets that
// cannot be written all at once.

import { Rate, Series, Synnax, TimeStamp, framer } from "@synnaxlabs/client";

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
}, { retrieveIfNameExists: true });

// Create a data channel that will be used to store our fake sensor data.
const dataChannel = await client.channels.create({
    name: "stream_write_example_data",
    dataType: "float32",
    index: timeChannel.key,
}, { retrieveIfNameExists: true });


// We'll start our write at the current time. This timestamps should be the same as or 
// just before the first timestamp we write.
const start = TimeStamp.now();

// Set a rough rate of 20 Hz. This won't be exact because we're sleeping for a fixed 
// amount of time, but it's close enough for demonstration purposes.
const roughRate = Rate.hz(40);

// Make the writer commit every 500 samples. This will make the data available for 
// historical reads every 500 samples.
const commitInterval = 500;

const writer = await client.telem.newWriter({
    start, 
    channels: [timeChannel.key, dataChannel.key]
});

try {
    let i = 0;
    while (true) {
        await new Promise(resolve => setTimeout(resolve, roughRate.period.milliseconds));
        i++;
        const timestamp = TimeStamp.now();
        const data = Math.sin(i / 10);
        const fr = new framer.Frame(
            [
                timeChannel.key, 
                dataChannel.key
            ],
            [
                new Series({ data: new timeChannel.dataType.Array([timestamp]) }),
                new Series({ data: new dataChannel.dataType.Array([data]) })
            ]
        );
        await writer.write(fr);

        if (i % 60 == 0) 
            console.log(`Writing sample ${i} at ${timestamp.toISOString()}`)

        if (i % commitInterval == 0) {
            // Commit the writer. This method will return false if the commit fails i.e.
            // we've mad an invalid write or someone has already written to this region.
            if (!await writer.commit()) {
                console.error("Failed to commit data");
                break;
            };
        }
    }
} finally {
    await writer.close();
}

