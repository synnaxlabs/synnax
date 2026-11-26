// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";
import { ZodError } from "zod";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";

const client = createTestClient();

// Creates a broker whose scan task the spec answers.
const createScannedBroker = async () => {
  const { key } = await createBroker(client);
  const dev = await client.devices.retrieve({ key, schemas: MQTT.Device.SCHEMAS });
  return { dev, scanner: await openScanner(client, dev.rack) };
};

describe("MQTT.Device browse", () => {
  it("should return the topics and the shapes of a browse reply", async () => {
    const { dev, scanner } = await createScannedBroker();
    const topic = {
      topic: "plant/temp",
      payload: '{"v": 21.5}',
      shapes: [
        {
          fields: [{ pointer: "/v", type: "number", kind: "float", sample: "21.5" }],
          count: 1,
        },
      ],
      retained: false,
    };
    const browsed = MQTT.Device.browseTopics(client, dev, "plant/#");
    await scanner.answer({ data: { topics: [topic], truncated: true } });
    expect(await browsed).toEqual({
      topics: [{ ...topic, key: "plant/temp" }],
      truncated: true,
    });
  });

  it("should fail a browse whose reply is not a browse reply", async () => {
    const { dev, scanner } = await createScannedBroker();
    const browsed = MQTT.Device.browseTopics(client, dev, "plant/#");
    await Promise.all([
      expect(browsed).rejects.toThrow(ZodError),
      scanner.answer({ data: { topics: [] } }),
    ]);
  });

  it("should fail a Sparkplug B browse whose reply is not a Sparkplug B reply", async () => {
    const { dev, scanner } = await createScannedBroker();
    const browsed = MQTT.Device.browseSparkplug(client, dev, {
      group: "plant",
      edgeNode: "line_1",
    });
    await Promise.all([
      expect(browsed).rejects.toThrow(ZodError),
      scanner.answer({ data: { nodes: [] } }),
    ]);
  });
});
