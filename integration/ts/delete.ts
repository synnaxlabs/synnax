import { Synnax } from "@synnaxlabs/client";
import { TimeRange } from "@synnaxlabs/x";
import { argv, exit } from "process";

class TestConfig{
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


async function deleteTest(tc: TestConfig): Promise<void>{
    client.delete(tc.channels, tc.timeRange);
}


function parseInput(argv: string[]): TestConfig {
    let argvCounter = 2
    const identifier = argv[argvCounter++];
    const timeRangeStart = BigInt(argv[argvCounter++]);
    const timeRangeEnd = BigInt(argv[argvCounter++]);
    const number_of_channels = parseInt(argv[argvCounter++]);
    let channels = [];
    for (let i = 0; i < number_of_channels; i++) {
        channels.push(argv[argvCounter++]);
    }

    return {
        identifier,
        timeRange: new TimeRange(timeRangeStart, timeRangeEnd),
        channels,
    };
}

async function main() {
    const tc = parseInput(argv);
    await deleteTest(tc).catch(e => {
        console.error(e)
        client.close()
        exit(1)
    })
    client.close()
}

await main()