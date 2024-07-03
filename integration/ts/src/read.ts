import { Synnax } from "@synnaxlabs/client";
import { Iterator } from "@synnaxlabs/client/dist/framer/iterator";
import { CrudeTimeStamp, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x"
import * as fs from 'fs'
import { argv } from 'process';

const FILE_NAME = "../../timing.log"

function approxEqual(a: number, b: number, tol = 0.01): boolean {
    return Math.abs(b - a) < tol * b;
}


class TestConfig {
    identifier: string = "";
    numIterators: number = 0;
    chunkSize: number = 1e5;
    bounds: TimeRange = TimeRange.ZERO;
    samplesExpected: number = 0;
    expectedError: string;
    channels: string[][] = [];

    constructor(
        identifier: string,
        numIterators: number,
        chunkSize: number,
        boundStart: CrudeTimeStamp,
        boundEnd: CrudeTimeStamp,
        samplesExpected: number,
        expectedError: string,
        channels: string[][],
    ) {
        this.identifier = identifier;
        this.numIterators = numIterators;
        this.chunkSize = chunkSize;
        this.bounds = new TimeRange(boundStart, boundEnd);
        this.samplesExpected = samplesExpected;
        this.expectedError = expectedError;
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
        const samplesExpected = parseInt(argv[argvCounter++]);
        const expectedError = argv[argvCounter++];
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
        this.tc = new TestConfig(identifier, numIterators, chunkSize, boundStart, boundEnd, samplesExpected, expectedError, channels);
    }

    async testWithTiming(): Promise<void> {
        const start = TimeStamp.now();
        let samples = 0;
        let errorAssertion = false;
        let actualError = "";
        let caught = false;
        await this.test().then(result => samples = result).catch((e: unknown) => {
            if (e instanceof Error) {
                caught = true;
                actualError = e.message;
                if (this.tc.expectedError != "no_error" && e.message.includes(this.tc.expectedError)) {
                    errorAssertion = true;
                } else {
                    throw e;
                }
            } else {
                throw e;
            }
        });

        if (!caught) {
            if (this.tc.expectedError == "no_error") {
                errorAssertion = true;
            }
            actualError = "no_error";
        }
        const end = TimeStamp.now();

        const time: TimeSpan = start.span(end);
        const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
        const assertionPassed = this.tc.samplesExpected == 0 || approxEqual(samples, this.tc.samplesExpected);
        const assertionResult = `Expected samples: ${this.tc.samplesExpected}; Actual samples: ${samples}`;
        const s = `
-- TypeScript Read (${this.tc.identifier}) --
Samples read: ${samples}
Time taken: ${time}
Calculated Samples per Second: ${samplesPerSecond.toFixed(2)}
Configuration:
\tNumber of iterators: ${this.tc.numIterators}
\tNumber of channels: ${this.tc.numChannels()}
\tChunk size: ${this.tc.chunkSize}
${assertionResult}: ${assertionPassed ? "PASS!!" : "FAIL!!"}
Expected error: ${this.tc.expectedError}; Actual error: ${actualError}\n${errorAssertion ? "PASS!!" : "FAIL!!"}
`;

        fs.appendFileSync(FILE_NAME, s);
    };

    async test(): Promise<number> {
        const iterators: Iterator[] = new Array(this.tc.numIterators).fill(null);
        let samples_read = 0;

        for (let i = 0; i < this.tc.numIterators; i++) {
            iterators[i] = await client.openIterator(
                this.tc.bounds,
                this.tc.channels[i],
                { chunkSize: this.tc.chunkSize },
            );
        }

        try {
            for (const i of iterators) {
                await i.seekFirst();
                for await (const frame of i) {
                    samples_read += frame.series.reduce((a, s) => a + s.length, 0);
                }
            }
        } finally {
            for (const i of iterators) {
                await i.close();
            }
        }

        return samples_read;
    }
}


async function main() {
    try {
        await new ReadTest(argv).testWithTiming();
    } finally {
        client.close();
    }
}

await main()
