// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const DRAWER = ".console-nav__drawer.pluto--visible:not(.pluto--location-bottom)";

/**
 * Docs `console/ui-overview/close-toolbars`: with the channels toolbar and the
 * visualization toolbar open over a streaming line plot, close the visualization
 * toolbar from its rail icon, then drag the channels drawer's edge into the rail
 * until it collapses.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    // The visualization toolbar stays open: it is the drawer the icon closes.
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.openToolbar(session, "channel");
    await capture.resizeToolbar(session, 320);
    // Live telemetry buffers on wall time: fill the rolling window before the
    // first frame so the plot never records as an empty grid.
    await session.settleWall(24000);
    await session.settle(1000);
    await session.moveTo({ x: 900, y: 420 });

    session.startRecording();
    await session.hold(1400);

    const y1 = page.locator("label").filter({ hasText: "Y1" }).first();
    // No zoom: closing a drawer resizes the whole window, so the camera stays wide.
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
    await session.waitForHidden(y1);
    // The rail's toggle slides down as the drawer closes, re-entering the parked
    // cursor and raising its tooltip. Move onto the next target before it does.
    const handle = page.locator(`${DRAWER} > .pluto-resize__handle`).first();
    await session.moveTo(handle);
    await session.hold(1200);

    // Release inside the rail, in the gap between the toolbar icons and the
    // visualization toggle: dwelling on an icon opens its drawer as a preview.
    const items = page.locator("button.console-main-nav__item");
    const count = await items.count();
    const above = await items.nth(count - 2).boundingBox();
    const below = await items.nth(count - 1).boundingBox();
    if (above == null || below == null) throw new Error("nav items have no box");
    const drawer = page.locator(DRAWER).first();
    await session.drag(
      handle,
      { x: 18, y: (above.y + above.height + below.y) / 2 },
      { zoom: false },
    );
    await session.waitForHidden(drawer);
    await session.hold(2400);
  } finally {
    await fixture.stop();
  }
};
