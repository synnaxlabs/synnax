import { Synnax } from "@synnaxlabs/client";
import { Iterator } from "@synnaxlabs/client/dist/framer/iterator";
import { CrudeTimeStamp, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x"
import * as fs from 'fs'
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

    numChannels(): number { return this.channels.reduce((a, l) => a + l.length, 0) }
}

const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false,
});


class ReadTest {
    tc: TestConfig;
    constructor(argv: string[]) {
        let argvCounter = 2
        const identifier = argv[argvCounter++];
        const numIterators = parseInt(argv[argvCounter++]);
        const chunkSize = parseInt(argv[argvCounter++]);
        const boundStart = BigInt(argv[argvCounter++]);
        const boundEnd = BigInt(argv[argvCounter++]);
        const number_of_channel_groups = parseInt(argv[argvCounter++]);
        const channels = [];
        for (let i = 0; i < number_of_channel_groups; i++) {
            const number_of_channels_in_group = parseInt(argv[argvCounter++]);
            const group = [];
            for (let j = 0; j < number_of_channels_in_group; j++) {
                group.push(argv[argvCounter++]);
            }
            channels.push(group);
        }
        this.tc = new TestConfig(identifier, numIterators, chunkSize, boundStart, boundEnd, channels);
    }

    async testWithTiming(): Promise<void> {
        const start = TimeStamp.now();
        const samples = await this.test();
        const end = TimeStamp.now();

        const time: TimeSpan = start.span(end);
        const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
        const s = `
-- TypeScript Read (${this.tc.identifier}) --
Samples read: ${samples}
Time taken: ${time}
Calculated Samples per Second: ${samplesPerSecond.toFixed(2)}
Configuration:
\tNumber of iterators: ${this.tc.numIterators}
\tNumber of channels: ${this.tc.numChannels()}
\tChunk size: ${this.tc.chunkSize}

`;

        fs.appendFileSync("../../timing.log", s);
    };

    async test(): Promise<number> {
        const iterators: Iterator[] = new Array(this.tc.numIterators).fill(null);
        let samples_read = 0;
        const start = TimeStamp.now();

        for (let i = 0; i < this.tc.numIterators; i++) {
            iterators[i] = await client.openIterator(
                this.tc.bounds,
                this.tc.channels[i],
                { chunkSize: this.tc.chunkSize },
            );
        }
        console.log("done creating", Number(start.span(TimeStamp.now()))/1000000)

        try {
            for (const i of iterators) {
                await i.seekFirst()
                console.log("done seeking", Number(start.span(TimeStamp.now()))/1000000)
                for await (const frame of i) {
                    samples_read += frame.series.reduce((a, s) => a + s.length, 0);
                }
                
                console.log("done reading", Number(start.span(TimeStamp.now()))/1000000)
            }
        } finally {
            for (const i of iterators) {
                await i.close()
            }
        }

        return samples_read
    }
}


async function main() {
    await new ReadTest(argv).testWithTiming().catch(e => {
        console.error(e)
        client.close()
        exit(1)
    })
    client.close()
}

await main()
