import { framer, Synnax } from "@synnaxlabs/client";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import * as fs from "fs";

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
  expectedError: string;
  channels: IndexWriterGroup[] = [];

  numChannels: () => number = () => {
    return this.channels.reduce((a, l: IndexWriterGroup) => a + l.together().length, 0);
  };

  constructor(
    identifier: string,
    numWriters: number,
    domains: number,
    samplesPerDomain: number,
    timeRangeStart: bigint,
    timeRangeEnd: bigint,
    autoCommit: boolean,
    indexPersistInterval: TimeSpan,
    writerMode: framer.WriterMode,
    expectedError: string,
    channels: IndexWriterGroup[],
  ) {
    this.identifier = identifier;
    this.numWriters = numWriters;
    this.domains = domains;
    this.samplesPerDomain = samplesPerDomain;
    this.timeRange = new TimeRange(timeRangeStart, timeRangeEnd);
    this.autoCommit = autoCommit;
    this.indexPersistInterval = indexPersistInterval;
    this.writerMode = writerMode;
    this.expectedError = expectedError;
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

class WriteTest {
  tc: TestConfig;
  constructor(argv: string[]) {
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
    const expectedError = argv[argvCounter++];
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

    this.tc = new TestConfig(
      identifier,
      numWriters,
      domains,
      samplesPerDomain,
      timeRangeStart,
      timeRangeEnd,
      autoCommit,
      indexPersistInterval,
      writerMode,
      expectedError,
      channelGroups,
    );
  }

  async testWithTiming(): Promise<void> {
    const start = TimeStamp.now();
    let errorAssertion = false;
    let actualError = "";
    let caught = false;

    await this.test().catch((e: unknown) => {
      if (e instanceof Error) {
        caught = true;
        actualError = e.message;
        if (
          this.tc.expectedError != "no_error" &&
          e.message.includes(this.tc.expectedError)
        ) {
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
    const samples = this.tc.numChannels() * this.tc.samplesPerDomain * this.tc.domains;
    const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
    const s = `
-- TypeScript Write (${this.tc.identifier}) --
Samples written: ${formatNumber(samples)}
Time taken: ${time}
Calculated Samples per Second: ${formatNumber(samplesPerSecond)}
Configuration:
\tNumber of writers: ${this.tc.numWriters}
\tNumber of channels: ${this.tc.numChannels()}
\tNumber of domains: ${formatNumber(this.tc.domains)}
\tSamples per domain: ${formatNumber(this.tc.samplesPerDomain)}
\tAuto commit: ${this.tc.autoCommit}
\tIndex persist interval: ${this.tc.indexPersistInterval}
\tWriter mode: ${framer.WriterMode[this.tc.writerMode]}

Expected error: ${this.tc.expectedError}; Actual error: ${actualError}\n${errorAssertion ? "PASS!!" : "FAIL!!"}
`;

    fs.appendFileSync("../../timing.log", s);
  }

  async test(): Promise<void> {
    const writers = new Array(this.tc.numWriters).fill(null);
    const timeSpanPerDomain = Number(this.tc.timeRange.span) / this.tc.domains;

    for (let i = 0; i < this.tc.numWriters; i++) {
      writers[i] = await client.openWriter({
        start: this.tc.timeRange.start,
        channels: this.tc.channels[i].together(),
        mode: this.tc.writerMode,
        enableAutoCommit: this.tc.autoCommit,
        autoIndexPersistInterval: this.tc.indexPersistInterval,
      });
    }

    try {
      let tsHwm = this.tc.timeRange.start;
      const timeSpanPerSample =
        BigInt(timeSpanPerDomain) / BigInt(this.tc.samplesPerDomain);
      for (let i = 0; i < this.tc.domains; i++) {
        const timestamps = Array.from<any, bigint>(
          { length: this.tc.samplesPerDomain },
          (_, k) => tsHwm.valueOf() + BigInt(k) * timeSpanPerSample,
        );
        const data = timestamps.map((ts) => Math.sin(0.000000001 * Number(ts)));
        tsHwm = tsHwm.add(timeSpanPerDomain);

        for (let j = 0; j < writers.length; j++) {
          const writer = writers[j];
          const dataDict: { [key: string]: number[] | bigint[] } = {};

          this.tc.channels[j].indexChannels.forEach((indexChannel) => {
            dataDict[indexChannel] = timestamps;
          });
          this.tc.channels[j].dataChannels.forEach((dataChannel) => {
            dataDict[dataChannel] = data;
          });

          await writer.write(dataDict);

          if (!this.tc.autoCommit) {
            await writer.commit();
          }

          await writer.close();

          writers[j] = await client.openWriter({
            start: tsHwm,
            channels: this.tc.channels[j].together(),
            mode: this.tc.writerMode,
            enableAutoCommit: this.tc.autoCommit,
            autoIndexPersistInterval: this.tc.indexPersistInterval,
            errOnUnauthorized: false,
          });
        }
      }
    } finally {
      for (const writer of writers) {
        await writer.close();
      }
    }
  }
}

function formatNumber(x: number): string {
  return x
    .toFixed(2)
    .toString()
    .replace(/\B(?<!\.\d*)(?=(\d{3})+(?!\d))/g, ",");
}

async function main(): Promise<void> {
  try {
    await new WriteTest(process.argv).testWithTiming();
  } finally {
    client.close();
  }
}

await main();
