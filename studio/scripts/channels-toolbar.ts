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
 * Docs `client/channels/toolbar`: with an empty line plot open on live data, open the
 * Channels toolbar, double-click tank_pressure onto the plot, drag tank_temperature
 * onto it, browse a channel's context menu, and group both channels as "Tank".
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry({ prefix: "tank" });
  try {
    const { page } = session;
    await capture.login(session);

    await capture.clearPanel(session);
    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Line plot");
    const plot = page.locator(".pluto-line-plot").first();
    await session.waitFor(plot);
    await capture.hideBottomToolbar(session);
    await capture.openToolbar(session, "channel");
    await capture.resizeToolbar(session, 400);
    await capture.closeToolbar(session);
    await session.settleWall(12_000);
    await session.moveTo({ x: 900, y: 450 });

    session.startRecording();
    await session.hold(1500);

    await capture.openToolbar(session, "channel");
    await session.hold(600);
    const pressure = capture.treeItem(page, "channel:", "tank_pressure");
    const temperature = capture.treeItem(page, "channel:", "tank_temperature");
    await session.waitFor(temperature);

    await session.doubleClick(pressure, { zoom: false, text: true });
    await session.hold(1800);
    // The plot takes drops on its Y axis, not on the canvas.
    const axis = plot.locator(".pluto-axis.lyra--location-left").first();
    await session.drag(temperature, axis, { zoom: false });
    await session.hold(2000);

    await session.rightClick(temperature, { text: true, zoom: false });
    const menu = page.locator(".lyra-menu-context").first();
    await session.waitFor(menu);
    await session.zoom([temperature, menu]);
    await session.hold(500);
    for (const item of await menu.locator(".lyra-menu-item").all()) {
      await session.moveTo(item);
      await session.hold(350);
    }
    await session.hold(400);
    await session.press("Escape");
    session.endZoom();
    await session.waitForHidden(menu);
    await session.hold(600);

    await session.click(pressure, { text: true });
    await page.keyboard.down("ControlOrMeta");
    await session.click(temperature, { text: true });
    await page.keyboard.up("ControlOrMeta");
    await session.hold(400);
    await capture.contextMenu(session, temperature, "Group selection");
    const editable = page
      .locator(".lyra-text--editable[contenteditable='true']")
      .first();
    await session.waitFor(editable);
    await session.hold(400);
    await session.type("Tank");
    await session.hold(300);
    await session.press("Enter");
    session.endZoom();
    await session.hold(2800);
  } finally {
    await fixture.stop();
  }
};
