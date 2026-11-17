// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { MQTT, serverTaskScript } from "./common/server";
import { setText } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

const record = serverTaskScript(MQTT, {
  item: "Create Sparkplug edge node",
  properties: async (session, form) => {
    await setText(session, form, "Group", "Plant");
    await setText(session, form, "Edge node", "Line2");
  },
  rates: { Authority: "200" },
  channels: async (session, form) => {
    const { page } = session;
    await session.click(form.getByRole("button", { name: "Add tag", exact: true }));
    const tag = form.getByRole("option").first();
    await session.waitFor(tag);
    await session.hold(600);
    await session.click(tag.locator(".lyra-dialog__trigger").first());
    await session.hold(400);
    await session.type("line2_temperature");
    const option = page
      .locator(".lyra-list__item:not(.lyra-tree__item)")
      .filter({ has: page.getByText("line2_temperature", { exact: true }) })
      .first();
    await session.waitFor(option);
    await session.hold(400);
    await session.click(option.getByText("line2_temperature", { exact: true }), {
      text: true,
    });
    await session.hold(800);
    await capture.contextMenu(session, tag, "Add command channel");
    await session.hold(800);
  },
});

/**
 * Docs `reference/hardware/mqtt/sparkplug` (Publish channels as an edge node): create an
 * edge node on the simulated broker, name it, publish a channel as a tag, let a host
 * write to the tag through a command channel, and start the task.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const telemetry = await fixtures.sineTelemetry({ prefix: "line2" });
  try {
    await record(session);
  } finally {
    await telemetry.stop();
  }
};
