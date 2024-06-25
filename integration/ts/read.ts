import { Synnax } from "@synnaxlabs/client";
import { AUTO_SPAN, Iterator } from "@synnaxlabs/client/dist/framer/iterator";
import { CrudeTimeStamp, TimeRange } from "@synnaxlabs/x"
import { argv, exit } from 'process';

class TestConfig {
    identifier: string = "";
    numIterators: number = 0;
    chunkSize: number = 1e5;
    bounds: TimeRange = TimeRange.ZERO;
    channels: string[][] = [];

    constructor(
        identifier: string,
        numIterators: number,
        chunkSize: number,
        boundStart: CrudeTimeStamp,
        boundEnd: CrudeTimeStamp,
        channels: string[][],
    ) {
        this.identifier = identifier;
        this.numIterators = numIterators;
        this.chunkSize = chunkSize;
        this.bounds = new TimeRange(boundStart, boundEnd);
        this.channels = channels;
    }

    numChannels: () => number = () => {
        return this.channels.reduce((a, l: string[]) => a + l.length, 0);
    };
}

const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false,
});


async function readTest(tc: TestConfig): Promise<number> {
    const iterators: Iterator[] = new Array(tc.numIterators).fill(null);
    let samples_read = 0;

    for (let i = 0; i < tc.numIterators; i++) {
        iterators[i] = await client.openIterator(
            tc.bounds,
            tc.channels[i],
            { chunkSize: tc.chunkSize },
        );
    }

    try {
        for (const i of iterators) {
            await i.seekFirst()
            for await (const frame of i) {
                samples_read += frame.series.reduce((a, s) => a + s.length, 0);
            }
        }
    } finally {
        for (const i of iterators) {
            await i.close()
        }
    }

    return samples_read
}

function parseInput(argv: string[]): TestConfig {
    let argvCounter = 2
    const identifier = argv[argvCounter++];
    const numIterators = parseInt(argv[argvCounter++]);
    const chunkSize = parseInt(argv[argvCounter++]);
    const boundStart = BigInt(argv[argvCounter++]);
    const boundEnd = BigInt(argv[argvCounter++]);
    const number_of_channel_groups = parseInt(argv[argvCounter++]);
    let channels = [];
    for (let i = 0; i < number_of_channel_groups; i++) {
        const number_of_channels_in_group = parseInt(argv[argvCounter++]);
        let group = [];
        for (let j = 0; j < number_of_channels_in_group; j++) {
            group.push(argv[argvCounter++]);
        }
        channels.push(group);
    }

    return new TestConfig(identifier, numIterators, chunkSize, boundStart, boundEnd, channels);
}


async function main() {
    const tc = parseInput(argv);
    await readTest(tc).catch(e => {
        console.error(e)
        client.close()
        exit(1)
    })
    client.close()
}

await main()
