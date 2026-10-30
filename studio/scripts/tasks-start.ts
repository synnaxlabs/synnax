// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni, type rack } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

const TASK = "Tank sensors";

/** Creates the NI analog read task the shot starts. */
const createTask = async (rackKey: rack.Key): Promise<void> => {
  const client = fixtures.connect();
  try {
    await client.tasks.create({
      name: TASK,
      type: "ni_analog_read",
      rack: rackKey,
      config: ni.analogReadConfigZ.parse({
        sampleRate: 100,
        streamRate: 10,
        channels: [0, 1].map((port) =>
          ni.aiVoltageChannelZ.parse({
            type: "ai_voltage",
            port,
            device: fixtures.NI_USB_6343.key,
            minVal: -10,
            maxVal: 10,
          }),
        ),
      }),
    });
  } finally {
    await client.close();
  }
};

/**
 * Docs `reference/hardware/start-and-stop-tasks` (Start a task): open the tasks
 * toolbar, double-click an NI analog read task to open its tab, and click Start; the
 * status turns to "Started task". The NI device is simulated.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const driver = await fixtures.simulatedDriver({
    devices: [
      fixtures.configured(fixtures.NI_USB_6343, fixtures.NI_USB_6343.name, "dev1"),
    ],
  });
  try {
    await createTask(driver.rack);

    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1500);

    await capture.openToolbar(session, "task");
    await session.hold(600);
    const item = page.locator(".console-task-toolbar").getByText(TASK, { exact: true });
    await session.waitFor(item);
    await session.doubleClick(item, { text: true });
    const form = page.locator(".console-task-configure").first();
    await session.waitFor(form);
    await session.hold(800);
    await session.zoom(form.locator(".console-task-controls").first(), 1.5);
    await session.hold(1200);

    await session.click(form.locator("button:has(svg.lyra-icon--play)").first());
    session.endZoom();
    await session.waitFor(form.getByText("Started task").first());
    await session.hold(3000);
  } finally {
    await driver.stop();
  }
};
