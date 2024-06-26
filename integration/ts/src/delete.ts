import { Synnax } from "@synnaxlabs/client";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import * as fs from 'fs';
import { argv, exit } from "process";

class TestConfig {
    identifier: string = "";
    channels: string[] = [];
    timeRange: TimeRange = TimeRange.ZERO;
}

const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false,
});

class DeleteTest {
    tc: TestConfig;
    constructor(argv: string[]) {
        let argvCounter = 2
        const identifier = argv[argvCounter++];
        const timeRangeStart = BigInt(argv[argvCounter++]);
        const timeRangeEnd = BigInt(argv[argvCounter++]);
        const number_of_channels = parseInt(argv[argvCounter++]);
        const channels = [];
        for (let i = 0; i < number_of_channels; i++) {
            channels.push(argv[argvCounter++]);
        }

        this.tc = {
            identifier,
            timeRange: new TimeRange(timeRangeStart, timeRangeEnd),
            channels,
        };
    }

    async testWithTiming(): Promise<void> {
        const start = TimeStamp.now();
        await this.test();
        const end = TimeStamp.now();

        const time: TimeSpan = start.span(end);
        const s = `
-- TypeScript Delete (${this.tc.identifier}) --
Time taken: ${time}
Configuration:
    Number of channels: ${this.tc.channels.length}
    Time Range: ${this.tc.timeRange}

`;

        fs.appendFileSync("timing.log", s);
    }

    async test(): Promise<void> {
        client.delete(this.tc.channels, this.tc.timeRange);
    }
}


async function main() {
    await new DeleteTest(argv).test().catch(e => {
        console.error(e)
        client.close()
        exit(1)
    })
    client.close()
}

await main()