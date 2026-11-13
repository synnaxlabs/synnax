// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type capture, fixtures } from "@/index";

import { KAFKA, serverTaskScript } from "./common/server";
import { addChannel, choose, setText } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

const record = serverTaskScript(KAFKA, {
  item: "Create write task",
  properties: async (session, form) => {
    await setText(session, form, "Topic", "synnax-samples");
  },
  channels: async (session, form) => {
    await addChannel(session, form);
    await choose(session, form, "Channel", "oven_temperature", { search: true });
    await choose(session, form, "Output type", "Number");
  },
});

/**
 * Docs `reference/hardware/kafka/write-task` (Create a Kafka write task): enter the
 * topic, add a channel, set its output type, and start the task, which publishes the
 * live channel to the fake broker.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const telemetry = await fixtures.sineTelemetry({ prefix: "oven" });
  try {
    await record(session);
  } finally {
    await telemetry.stop();
  }
};
