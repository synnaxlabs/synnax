// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, type Synnax, task } from "@synnaxlabs/client";
import { deep, TimeStamp } from "@synnaxlabs/x";

import { MAKE, type Properties, ZERO_PROPERTIES } from "@/feature/mqtt/device/types";
import { SCAN_TYPE } from "@/feature/mqtt/task/types";
import { createTestDevice } from "@/platform/device/testutil";
import { awaitCommand } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

export interface CreateBrokerOptions {
  configured?: boolean;
  properties?: Partial<Properties>;
}

/** Creates a rack and a configured MQTT broker device on the live Core. */
export const createBroker = async (
  client: Synnax,
  { configured = true, properties }: CreateBrokerOptions = {},
): Promise<device.Device> =>
  await createTestDevice(client, {
    name: uniqueName("mqtt_broker"),
    make: MAKE,
    model: "MQTT broker",
    configured,
    properties: { ...deep.copy(ZERO_PROPERTIES), ...properties },
  });

export interface ScannerReply {
  variant?: "success" | "error";
  message?: string;
  data?: object;
}

export interface Scanner {
  /** Answers the next command of the scan task, and returns that command. */
  answer: (reply: ScannerReply) => Promise<task.Command>;
  close: () => void;
}

/**
 * Creates a scan task on the rack and answers its commands in place of the driver.
 * Open it before the command goes out.
 */
export const openScanner = async (
  client: Synnax,
  rackKey: rack.Key,
): Promise<Scanner> => {
  const rck = await client.racks.retrieve({ key: rackKey });
  const scanTask = await rck.createTask({
    name: "MQTT scanner",
    type: SCAN_TYPE,
    config: {},
  });
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  return {
    answer: async ({ variant = "success", message = "", data }) => {
      const cmd = await awaitCommand(streamer, scanTask.key);
      await client.statuses.set({
        key: task.statusKey(scanTask.key),
        name: scanTask.name,
        variant,
        message,
        time: TimeStamp.now(),
        details: { task: scanTask.key, running: true, cmd: cmd.key, data },
      });
      return cmd;
    },
    close: () => streamer.close(),
  };
};
