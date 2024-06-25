import { TimeRange, TimeStamp, TimeSpan } from "@synnaxlabs/x";
import { Synnax, framer } from "@synnaxlabs/client"
import { exit } from "process";
import { TimeWrite } from "timing";

const client = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
});

class IndexWriterGroup {
  indexChannels: string[] = [];
  dataChannels: string[] = [];

  constructor(indexChannels: string[], dataChannels: string[]) {
    this.indexChannels = indexChannels;
    this.dataChannels = dataChannels;
  }

  together(): string[] {
    return [...this.indexChannels, ...this.dataChannels];
  }
}

class TestConfig {
  identifier: string = "";
  numWriters: number = 0;
  domains: number = 0;
  samplesPerDomain: number = 0;
  timeRange: TimeRange = TimeRange.ZERO;
  autoCommit: boolean = false;
  indexPersistInterval: TimeSpan = TimeSpan.seconds(1);
  writerMode: framer.WriterMode = framer.WriterMode.PersistStream;
  channels: IndexWriterGroup[] = [];
}

class WriteTest {
  async test(tc: TestConfig): Promise<void> {
    const writers = new Array(tc.numWriters).fill(null);
    const timeSpanPerDomain = Number(tc.timeRange.span) / tc.domains;

    for (let i = 0; i < tc.numWriters; i++) {
      writers[i] = await client.openWriter({
        start: tc.timeRange.start,
        channels: tc.channels[i].together(),
        mode: tc.writerMode,
        enableAutoCommit: tc.autoCommit,
        autoIndexPersistInterval: tc.indexPersistInterval,
      });
    }

    try {
      let tsHwm = tc.timeRange.start.add(new TimeSpan(1));
      for (let i = 0; i < tc.domains; i++) {
        const timestamps = Array.from<any, bigint>(
          { length: tc.samplesPerDomain },
          (_, k) => tsHwm.valueOf() + BigInt(k * timeSpanPerDomain) / BigInt(tc.samplesPerDomain));
        const data = timestamps.map(ts => Math.sin(0.0000000001 * Number(ts)));

        for (let j = 0; j < writers.length; j++) {
          const writer = writers[j];
          const dataDict: { [key: string]: number[] | bigint[] } = {};

          tc.channels[j].indexChannels.forEach(indexChannel => {
            dataDict[indexChannel] = timestamps;
          });
          tc.channels[j].dataChannels.forEach(dataChannel => {
            dataDict[dataChannel] = data;
          });

          await writer.write(dataDict);

          if (!tc.autoCommit) {
            await writer.commit();
          }
        }

        tsHwm.add(new TimeSpan(timeSpanPerDomain + 1));
      }
    } finally {
      for (const writer of writers) {
        await writer.close();
      }
    }
  }
}

function parseInput(argv: string[]): TestConfig {
  let argvCounter = 2;
  const identifier = argv[argvCounter++];
  const numWriters = parseInt(argv[argvCounter++]);
  const domains = parseInt(argv[argvCounter++]);
  const samplesPerDomain = parseInt(argv[argvCounter++]);
  const timeRangeStart = BigInt(argv[argvCounter++]);
  const timeRangeEnd = BigInt(argv[argvCounter++]);
  const autoCommit = argv[argvCounter++] === "true";
  const indexPersistInterval = new TimeSpan(BigInt(argv[argvCounter++]));
  const writerMode = parseInt(argv[argvCounter++]) as framer.WriterMode;
  const numberOfChannelGroups = parseInt(argv[argvCounter++]);
  const channelGroups: IndexWriterGroup[] = [];

  for (let i = 0; i < numberOfChannelGroups; i++) {
    const numberOfIndex = parseInt(argv[argvCounter++]);
    const numberOfData = parseInt(argv[argvCounter++]);
    const indexChannels = argv.slice(argvCounter, argvCounter + numberOfIndex);
    argvCounter += numberOfIndex;
    const dataChannels = argv.slice(argvCounter, argvCounter + numberOfData);
    argvCounter += numberOfData;
    channelGroups.push(new IndexWriterGroup(indexChannels, dataChannels));
  }

  return {
    identifier,
    numWriters,
    domains,
    samplesPerDomain,
    timeRange: new TimeRange(timeRangeStart, timeRangeEnd),
    autoCommit,
    indexPersistInterval,
    writerMode,
    channels: channelGroups,
  };
}

async function main(): Promise<void> {
  const tc = parseInput(process.argv);
  await new WriteTest().test(tc).catch(error => {
    console.error(error)
    client.close()
    exit(1)
  })

  client.close()
}

await main()
