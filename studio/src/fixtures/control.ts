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

export interface ValveOptions extends ConnectionOptions {
  /** Prefix of the created channel names. */
  name?: string;
  /** Interval between state samples, so the symbol never goes stale. */
  periodMs?: number;
}

export interface ValveFixture {
  /** Channel the schematic symbol writes commands to. */
  command: string;
  /** Channel the schematic symbol reads its displayed state from. */
  state: string;
  /** Stops the echo loop and closes the streamer, writer, and client. */
  stop: () => Promise<void>;
}

/**
 * echoValve creates a command/state channel pair and mirrors every command
 * back onto the state channel, so a schematic actuator bound to the pair
 * latches when it is clicked. Without the echo the symbol springs back: it
 * displays the state channel, which nothing else drives. The state is written
 * every `periodMs` whether or not it changed.
 */
export const echoValve = async ({
  name = "valve",
  periodMs = 40,
  ...opts
}: ValveOptions = {}): Promise<ValveFixture> => {
  const client = connect(opts);
  const create = async (channel: string, isIndex: boolean, index?: number) =>
    await client.channels.create(
      isIndex
        ? { name: channel, isIndex: true, dataType: "timestamp" }
        : { name: channel, dataType: "uint8", index },
      { retrieveIfNameExists: true },
    );
  const commandTime = await create(`${name}_cmd_time`, true);
  const command = await create(`${name}_cmd`, false, commandTime.key);
  const stateTime = await create(`${name}_state_time`, true);
  const state = await create(`${name}_state`, false, stateTime.key);

  const writer = await client.openWriter({
    start: TimeStamp.now(),
    channels: [stateTime.key, state.key],
  });

  let current = 0;
  const streamer = await client.openStreamer([command.key]);
  const echo = (async () => {
    for await (const frame of streamer) {
      const series = frame.get(command.key);
      if (series.length > 0) current = Number(series.at(-1));
    }
  })();

  let running = true;
  const heartbeat = (async () => {
    while (running) {
      await writer.write({ [stateTime.key]: TimeStamp.now(), [state.key]: current });
      await new Promise((resolve) => setTimeout(resolve, periodMs));
    }
    await writer.close();
  })();

  return {
    command: command.name,
    state: state.name,
    stop: async () => {
      running = false;
      streamer.close();
      await Promise.all([echo, heartbeat]);
      await client.close();
    },
  };
};
