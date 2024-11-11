import { Synnax, TimeSpan, TimeStamp } from "@synnaxlabs/client";
import * as fs from "fs";
import { argv } from "process";

class TestConfig {
  identifier: string;
  startTimeStamp: TimeStamp;
  samplesExpected: number;
  expectedError: string;
  channels: string[];

  constructor(
    identifier: string,
    startTimeStamp: bigint,
    samplesExpected: number,
    expectedError: string,
    channels: string[],
  ) {
    this.identifier = identifier;
    this.startTimeStamp = new TimeStamp(startTimeStamp);
    this.samplesExpected = samplesExpected;
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

class StreamTest {
  tc: TestConfig;

  constructor(argv: string[]) {
    let argvCounter = 2;
    const identifier = argv[argvCounter++];
    const startTimeStamp = BigInt(argv[argvCounter++]);
    const samplesExpected = parseInt(argv[argvCounter++]);
    const expectedError = argv[argvCounter++];
    const numberOfChannels = parseInt(argv[argvCounter++]);

    const channels: string[] = [];
    for (let i = 0; i < numberOfChannels; i++) channels.push(argv[argvCounter++]);

    this.tc = new TestConfig(
      identifier,
      startTimeStamp,
      samplesExpected,
      expectedError,
      channels,
    );
  }

  async testWithTiming(): Promise<void> {
    const start = TimeStamp.now();
    let errorAssertion = false;
    let actualError = "";
    let samples = 0;
    let caught = false;
    await this.test()
      .then((result) => {
        samples = result;
      })
      .catch((e: unknown) => {
        if (e instanceof Error) {
          caught = true;
          actualError = e.message;
          if (
            this.tc.expectedError != "no_error" &&
            e.message.includes(this.tc.expectedError)
          )
            errorAssertion = true;
          else throw e;
        } else throw e;
      });
    if (!caught) {
      if (this.tc.expectedError == "no_error") errorAssertion = true;
      actualError = "no_error";
    }

    const end = TimeStamp.now();

    const time: TimeSpan = start.span(end);
    const samplesPerSecond = samples / (Number(time) / Number(TimeSpan.SECOND));
    const s = `
-- TypeScript Stream (${this.tc.identifier}) --
Samples streamed: ${formatNumber(samples)}
Time taken: ${time}
Calculated Samples per Second: ${formatNumber(samplesPerSecond)}
Configuration:
\tNumber of streamers: 1
\tNumber of channels: ${this.tc.channels.length}
\tSamples expected: ${formatNumber(this.tc.samplesExpected)}

Expected error: ${this.tc.expectedError}; Actual error: ${actualError}\n${errorAssertion ? "PASS!!" : "FAIL!!!!"}
`;

    fs.appendFileSync("../../timing.log", s);
  }

  async test(): Promise<number> {
    let samplesStreamed = 0;

    const streamer = await client.openStreamer({
      channels: this.tc.channels,
    });

    try {
      for await (const frame of streamer) {
        samplesStreamed += frame.series.reduce((total, s) => total + s.length, 0);
        if (samplesStreamed >= 0.95 * this.tc.samplesExpected) return samplesStreamed;
      }
    } finally {
      streamer.close();
    }

    return samplesStreamed;
  }
}

function formatNumber(x: number): string {
  return x
    .toFixed(2)
    .toString()
    .replace(/\B(?<!\.\d*)(?=(\d{3})+(?!\d))/g, ",");
}

async function main() {
  try {
    await new StreamTest(argv).testWithTiming();
  } finally {
    client.close();
  }
}

await main();
