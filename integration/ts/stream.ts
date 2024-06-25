import { Synnax, TimeStamp } from "@synnaxlabs/client";
import { argv, exit } from 'process';
import { TimeStream } from "./timing";

class TestConfig {
    identifier: string;
    startTimeStamp: TimeStamp;
    closeAfterFrames: number;
    channels: string[];

    constructor(identifier: string, startTimeStamp: bigint, closeAfterFrames: number, channels: string[]) {
        this.identifier = identifier;
        this.startTimeStamp = new TimeStamp(startTimeStamp);
        this.closeAfterFrames = closeAfterFrames;
        this.channels = channels;
    }
}

const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false,
});

class StreamTest {
    @TimeStream
    async test(tc: TestConfig): Promise<number> {
        let counter = 0;
        let samplesStreamed = 0;

        const streamer = await client.openStreamer({
            channels: tc.channels,
            from: tc.startTimeStamp,
        });

        console.log(tc.channels)
        const f = await streamer.read()
        console.log(f)
        try {
            for await (const frame of streamer) {
                counter += 1;
                if (counter >= tc.closeAfterFrames) {
                    return samplesStreamed;
                }
                console.log(frame)
                samplesStreamed += frame.series.reduce((total, series) => total + series.length, 0);
            }
        } finally {
            streamer.close();
        }

        return samplesStreamed;
    }
}

function parseInput(argv: string[]): TestConfig {
    let argvCounter = 2;
    const identifier = argv[argvCounter++];
    const startTimeStamp = BigInt(argv[argvCounter++]);
    const closeAfterFrames = parseInt(argv[argvCounter++]);
    const numberOfChannels = parseInt(argv[argvCounter++]);

    const channels: string[] = [];
    for (let i = 0; i < numberOfChannels; i++) {
        channels.push(argv[argvCounter++]);
    }

    return new TestConfig(identifier, startTimeStamp, closeAfterFrames, channels);
}

async function main() {
    const tc = parseInput(argv);
    await new StreamTest().test(tc).catch(e => {
        console.error(e);
        client.close();
        exit(1);
    });
    client.close();
}

// await main()


// We can just specify the names of the channels we'd like to stream from.
const read_from = [
    "int0"
]
console.log(read_from)

const streamer = await client.openStreamer(read_from);

// It's very important that we close the streamer when we're done with it to release
// network connections and other resources, so we wrap the streaming loop in a try-finally
// block.
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

