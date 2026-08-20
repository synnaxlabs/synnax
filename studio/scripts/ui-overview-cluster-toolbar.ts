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
 * Docs `console/ui-overview/cluster-toolbar`: the connection badge in the top
 * right names the Core the Console works against, reports its state and
 * version, and opens the dialog holding its address.
 *
 * The Cores list, and with it switching the active Core, is Tauri-only: the
 * browser build detects the Core that serves it, so the login surface goes
 * straight to credentials and never offers the list.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });
    // The capture clock is a fixed epoch days behind the Core, so the badge
    // reports a clock skew that no user sees.
    await page.addStyleTag({
      content:
        ".console-connection-badge__body .pluto--status-warning " +
        "{ display: none !important; }",
    });

    // A live plot behind the dialog: the badge belongs to a working Console,
    // and an empty panel leaves half the frame blank.
    await capture.clearPanel(session);
    await capture.clickPanelCreate(session);
    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);
    await session.settleWall(20000);
    await session.settle(1000);
    // Off the plot: the cursor draws a crosshair and a value tooltip on it.
    await session.moveTo({ x: 700, y: 62 });

    session.startRecording();
    await session.hold(1200);

    const badge = page
      .locator("button")
      .filter({ has: page.locator(".console-connection-indicator") })
      .last();
    await session.click(badge, { zoom: false });
    const dialog = page.locator(".console-connection-badge__dialog").first();
    await session.waitFor(dialog);
    await session.hold(600);
    await session.zoom(dialog, 1.9);
    await session.hold(2800);
    session.endZoom();
    await session.hold(500);

    await session.click(page.getByRole("button", { name: "Edit connection" }).first(), {
      zoom: false,
    });
    const modal = page.locator(".console-modal").first();
    await session.waitFor(modal);
    await session.hold(600);
    await session.zoom(modal);
    await session.hold(2800);
  } finally {
    await fixture.stop();
  }
};
