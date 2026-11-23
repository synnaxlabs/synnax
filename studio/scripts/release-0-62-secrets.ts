// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { pagerduty, type secret, type Synnax } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

export const viewport = { width: 1280, height: 800 };

const ON_CALL = "TPC on-call routing key";
/** Panels removed so the top bar shows every panel's full name. */
const UNSHOWN = new Set(["Automations", "Fuel", "Hardware"]);

/** createAlertTask creates a PagerDuty alert task that pages through routingKey. */
const createAlertTask = async (
  client: Synnax,
  rack: number,
  name: string,
  routingKey: secret.Key,
): Promise<void> => {
  await client.tasks.create({
    name,
    type: "pagerduty_alert",
    rack,
    config: pagerduty.taskConfigZ.parse({ routingKey }),
  });
};

/**
 * Still for the Settings section of the v0.62 release notes: the Secrets page of the
 * TPC project, with the popover that lists the two alert tasks that page through the
 * on-call routing key.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const [onCall, sift, facilities] = await client.secrets.set(
      [
        ON_CALL,
        "Sift API key",
        "Facilities routing key",
        "Historian API token",
        "Kafka SASL password",
        "OPC UA server password",
        "Weather station token",
      ].map((name) => ({
        source: "stored" as const,
        name,
        value: crypto.randomUUID(),
      })),
    );
    const [pagerdutyRack] = await client.racks.retrieve({ integrations: "pagerduty" });
    await createAlertTask(client, pagerdutyRack.key, "Oxidizer alerts", onCall.key);
    await createAlertTask(client, pagerdutyRack.key, "Pressurant alerts", onCall.key);
    await createAlertTask(client, pagerdutyRack.key, "Facility alerts", facilities.key);
    const [siftRack] = await client.racks.retrieve({ integrations: "sift" });
    await client.devices.create({
      key: "sift",
      name: "Sift",
      make: "sift",
      model: "sift",
      location: "grpc-api.siftstack.com:443",
      rack: siftRack.key,
      configured: true,
      properties: { uri: "grpc-api.siftstack.com:443", apiKey: sift.key },
    });
    const panels = await client.panels.retrieve({});
    await client.panels.delete(
      panels.filter((p) => UNSHOWN.has(p.name)).map((p) => p.key),
    );

    await capture.login(session, world.project);
    await capture.openSettings(session, "Secrets");
    const item = page.getByRole("option", { name: ON_CALL, exact: true });
    await session.waitFor(item.getByText("Used by 2", { exact: true }));
    await session.click(item.getByText("Used by 2", { exact: true }), { zoom: false });
    await session.waitFor(page.getByText("Pressurant alerts", { exact: true }).last());
    await session.moveTo({ x: 26, y: 600 });
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
