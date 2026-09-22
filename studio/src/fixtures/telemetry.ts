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

export interface TelemetryOptions extends ConnectionOptions {
  /** Wall-clock milliseconds between samples. */
  periodMs?: number;
}

export interface TelemetryFixture {
  /** Names of the created data channels, for selection in scripts. */
  channels: string[];
  /** Stops the stream and closes the writer and client. */
  stop: () => Promise<void>;
}

/**
 * sineTelemetry creates demo channels on the cluster and streams synthetic
 * waveforms to them until stopped, so plots in a shot have live data to draw.
 * Streaming runs on wall time: the app receives frames in real time while the
 * capture clock steps, so on-screen data progresses faster than video time.
 */
export const sineTelemetry = async ({
  periodMs = 40,
  ...opts
}: TelemetryOptions = {}): Promise<TelemetryFixture> => {
  const client = connect(opts);
  const time = await client.channels.create(
    { name: "demo_time", isIndex: true, dataType: "timestamp" },
    { retrieveIfNameExists: true },
  );
  const pressure = await client.channels.create(
    { name: "demo_pressure", dataType: "float32", index: time.key },
    { retrieveIfNameExists: true },
  );
  const temperature = await client.channels.create(
    { name: "demo_temperature", dataType: "float32", index: time.key },
    { retrieveIfNameExists: true },
  );

  const writer = await client.openWriter({
    start: TimeStamp.now(),
    channels: [time.key, pressure.key, temperature.key],
  });

  let running = true;
  const loop = (async () => {
    let i = 0;
    while (running) {
      await new Promise((resolve) => setTimeout(resolve, periodMs));
      i++;
      await writer.write({
        [time.key]: TimeStamp.now(),
        [pressure.key]: 60 + 25 * Math.sin(i / 25) + 2 * Math.sin(i / 3),
        [temperature.key]: 21 + 8 * Math.cos(i / 40) + 0.4 * Math.sin(i / 5),
      });
    }
    await writer.close();
    await client.close();
  })();

  return {
    channels: [pressure.name, temperature.name],
    stop: async () => {
      running = false;
      await loop;
    },
  };
};
