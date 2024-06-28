import { Synnax, TimeSpan, TimeStamp } from "@synnaxlabs/client";
import * as fs from 'fs';
import { argv, exit } from 'process';

class TestConfig {
    identifier: string;
    startTimeStamp: TimeStamp;
    samplesExpected: number;
    channels: string[];

    constructor(identifier: string, startTimeStamp: bigint, samplesExpected: number, channels: string[]) {
        this.identifier = identifier;
        this.startTimeStamp = new TimeStamp(startTimeStamp);
        this.samplesExpected = samplesExpected;
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
    tc: TestConfig

    constructor(argv: string[]) {
        let argvCounter = 2;
        const identifier = argv[argvCounter++];
        const startTimeStamp = BigInt(argv[argvCounter++]);
        const samplesExpected = parseInt(argv[argvCounter++]);
        const numberOfChannels = parseInt(argv[argvCounter++]);

        const channels: string[] = [];
        for (let i = 0; i < numberOfChannels; i++) {
            channels.push(argv[argvCounter++]);
        }

        this.tc = new TestConfig(identifier, startTimeStamp, samplesExpected, channels);
    }

    async testWithTiming(): Promise<void> {
        const start = TimeStamp.now();
        const samples = await this.test();
        const end = TimeStamp.now();

        const time: TimeSpan = start.span(end);
        const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
        const s = `
-- TypeScript Stream (${this.tc.identifier}) --
Samples streamed: ${samples}
Time taken: ${time}
Calculated Samples per Second: ${samplesPerSecond.toFixed(2)}
Configuration:
\tNumber of streamers: 1
\tNumber of channels: ${this.tc.channels.length}
\tSamples expected: ${this.tc.samplesExpected}

`;

        fs.appendFileSync("../../timing.log", s);
    };

    async test(): Promise<number> {
        let samplesStreamed = 0;

        const streamer = await client.openStreamer({
            channels: this.tc.channels,
            from: this.tc.startTimeStamp,
        });

        try {
            for await (const frame of streamer) {
                samplesStreamed += frame.series.reduce((total, s) => total + s.length, 0);
                console.log(samplesStreamed)
                if (samplesStreamed >= this.tc.samplesExpected) {
                    return samplesStreamed;
                }
            }
        } finally {
            streamer.close();
        }

        return samplesStreamed;
    }
}

async function main() {
    await new StreamTest(argv).testWithTiming().catch(e => {
        console.error(e);
        client.close();
        exit(1);
    });
    client.close();
}

await main()
