// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

/**
 * Docs `client/channels/delete`: select tank_pressure and tank_temperature in the
 * Channels toolbar, right-click, select "Delete", and confirm. Both channels leave
 * the toolbar.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry({ prefix: "tank" });
  await fixture.stop();
  const { page } = session;
  await capture.login(session);

  await capture.clearPanel(session);
  await capture.openToolbar(session, "channel");
  await capture.resizeToolbar(session, 400);
  const pressure = capture.treeItem(page, "channel:", "tank_pressure");
  const temperature = capture.treeItem(page, "channel:", "tank_temperature");
  await session.waitFor(temperature);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await session.click(pressure, { text: true });
  await page.keyboard.down("ControlOrMeta");
  await session.click(temperature, { text: true });
  await page.keyboard.up("ControlOrMeta");
  await session.hold(500);
  await capture.contextMenu(session, temperature, "Delete");

  const dialog = page.locator(".console-confirm").first();
  await session.waitFor(dialog);
  await session.hold(400);
  await session.zoom(dialog, 1.6);
  await session.hold(800);
  await session.click(dialog.getByRole("button", { name: "Delete", exact: true }));
  await session.waitForHidden(dialog);
  session.endZoom();
  await session.waitForHidden(temperature);
  await session.hold(2800);
};
