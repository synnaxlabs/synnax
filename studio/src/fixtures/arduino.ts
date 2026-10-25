// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/client";

import { connect, type ConnectionOptions } from "@/fixtures/client";

export interface ArduinoOptions extends ConnectionOptions {
  /** Interval between samples. */
  periodMs?: number;
}

export interface ArduinoFixture {
  /** Virtual channel that the switch writes commands to. */
  command: string;
  /** Channel that echoes the last command, as the device reports it. */
  state: string;
  /** Channel of analog input readings. */
  value: string;
  /** Stops the loops and closes the streamer, writer, and client. */
  stop: () => Promise<void>;
}

/**
 * arduino stands in for the custom devices tutorial: the channels its scripts create,
 * and the samples its read-write script writes. It records an analogRead-like value,
 * and echoes each command on `arduino_command` back onto `arduino_state`.
 */
export const arduino = async ({
  periodMs = 40,
  ...opts
}: ArduinoOptions = {}): Promise<ArduinoFixture> => {
  const client = connect(opts);
  const command = await client.channels.create(
    { name: "arduino_command", dataType: "uint8", virtual: true },
    { retrieveIfNameExists: true },
  );
  const time = await client.channels.create(
    { name: "arduino_time", isIndex: true, dataType: "timestamp" },
    { retrieveIfNameExists: true },
  );
  const [state, value] = await client.channels.create(
    [
      { name: "arduino_state", dataType: "uint8", index: time.key },
      { name: "arduino_value", dataType: "float32", index: time.key },
    ],
    { retrieveIfNameExists: true },
  );

  let current = 0;
  const streamer = await client.openStreamer([command.key]);
  const echo = (async () => {
    for await (const frame of streamer) {
      const series = frame.get(command.key);
      if (series.length > 0) current = Number(series.at(-1));
    }
  })();

  const writer = await client.openWriter({
    start: TimeStamp.now(),
    channels: [time.key, state.key, value.key],
  });
  let running = true;
  const record = (async () => {
    let i = 0;
    while (running) {
      await new Promise((resolve) => setTimeout(resolve, periodMs));
      i++;
      const reading = 512 + 260 * Math.sin(i / 30) + 18 * Math.sin(i / 2.3);
      await writer.write({
        [time.key]: TimeStamp.now(),
        [state.key]: current,
        [value.key]: Math.round(reading),
      });
    }
    await writer.close();
  })();

  return {
    command: command.name,
    state: state.name,
    value: value.name,
    stop: async () => {
      running = false;
      streamer.close();
      await Promise.all([echo, record]);
      await client.close();
    },
  };
};
