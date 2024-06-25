import { Synnax, DataType } from "@synnaxlabs/client";
import { argv, exit } from 'process';

class SetUpConfig {
    numIndex: number;
    numData: number;

    constructor(numIndex: number, numData: number) {
        this.numIndex = numIndex;
        this.numData = numData;
    }
}

const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false,
});

async function createChannels(tc: SetUpConfig): Promise<void> {
    const channels: number[] = [];

    for (let i = 0; i < tc.numIndex; i++) {
        const index = await client.channels.create({
            name: `ch${i}`,
            isIndex: true,
            dataType: DataType.TIMESTAMP,
        }, { retrieveIfNameExists: true });
        channels.push(index.key);
    }

    const numDataChannelsPerIndex = Math.floor(tc.numData / tc.numIndex);
    for (let ind = 0; ind < tc.numIndex; ind++) {
        for (let k = 0; k < numDataChannelsPerIndex; k++) {
            const ch = await client.channels.create({
                name: `ch${ind}-${k}`,
                index: channels[ind],
                dataType: DataType.FLOAT32,
            }, { retrieveIfNameExists: true });
            channels.push(ch.key);
        }
    }
}

async function main() {
    if (argv.length !== 4) {
        console.error("Usage: node setup.ts <num_index> <num_data>");
        exit(-1);
    }

    const numIndex = parseInt(argv[2], 10);
    const numData = parseInt(argv[3], 10);
    const tc = new SetUpConfig(numIndex, numData);
    await createChannels(tc).catch(error => {
        console.error(error);
        client.close()
        exit(1)
    })

    client.close()
}

main()
