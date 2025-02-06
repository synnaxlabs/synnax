// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";
import { TimeRange, type TimeSpan, TimeStamp } from "@synnaxlabs/x";
import * as fs from "fs";
import { argv } from "process";

class TestConfig {
  identifier: string = "";
  expectedError: string = "";
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
    let argvCounter = 2;
    const identifier = argv[argvCounter++];
    const timeRangeStart = BigInt(argv[argvCounter++]);
    const timeRangeEnd = BigInt(argv[argvCounter++]);
    const expectedError = argv[argvCounter++];
    const number_of_channels = parseInt(argv[argvCounter++]);
    const channels = [];
    for (let i = 0; i < number_of_channels; i++) channels.push(argv[argvCounter++]);

    this.tc = {
      identifier,
      expectedError,
      timeRange: new TimeRange(timeRangeStart, timeRangeEnd),
      channels,
    };
  }

  async testWithTiming(): Promise<void> {
    const start = TimeStamp.now();
    let errorAssertion = false;
    let actualError = "";
    let caught = false;
    await this.test().catch((e: unknown) => {
      console.log("CAUGHT: ", e);
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
    const s = `
-- TypeScript Delete (${this.tc.identifier}) --
Time taken: ${time.isZero ? 0 : time}
Configuration:
\tNumber of channels: ${this.tc.channels.length}

Expected error: ${this.tc.expectedError}; Actual error: ${actualError}: ${errorAssertion ? "PASS!!" : "FAIL!!!!"}
`;

    fs.appendFileSync("../../timing.log", s);
  }

  async test(): Promise<void> {
    client.delete(this.tc.channels, this.tc.timeRange);
  }
}

async function main() {
  try {
    await new DeleteTest(argv).testWithTiming();
  } finally {
    client.close();
  }
}

await main();
