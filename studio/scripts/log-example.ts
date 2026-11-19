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
 * Docs `console/logs/example`: with a log open, open its toolbar, add streaming
 * channels in the Channels tab, and set the precision of one; lines arrive as
 * telemetry does.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Log");
    await session.waitFor(page.locator(".pluto-log").first());
    await capture.hideBottomToolbar(session);
    // Off the log's "Configure channels" action.
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    const drawer = page.locator(".console-nav__drawer.lyra--location-bottom").first();
    const add = page.getByText("Add channel", { exact: true }).first();
    if (!(await add.isVisible().catch(() => false)))
      await session.click(page.locator(".console-main-nav__item").last(), {
        zoom: false,
      });
    await session.waitFor(add);
    await session.hold(600);
    await capture.zoomCenter(session, drawer, 1.4);

    for (const channel of fixture.channels) {
      await session.click(page.getByText("Add channel", { exact: true }).first(), {
        text: true,
        zoom: false,
      });
      const input = page.locator("input[placeholder*='Search']");
      await session.waitFor(input);
      await session.hold(400);
      await session.type(channel);
      const item = page
        .locator(".lyra-list__item:not(.lyra-tree__item)")
        .filter({ hasText: channel })
        .first();
      await session.waitFor(item);
      await session.click(item.getByText(channel, { exact: true }).first(), {
        text: true,
      });
      await session.hold(800);
    }

    const precision = drawer.locator(".console-log__channel-precision input").first();
    await session.click(precision);
    await session.type("2");
    await session.press("Enter");
    session.endZoom();
    await session.moveTo({ x: 1200, y: 300 });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
