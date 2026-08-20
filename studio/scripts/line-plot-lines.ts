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
 * Docs `console/line-plots/lines-tab`: in the Lines tab, relabel a line, widen
 * its stroke, and recolor it; the legend and the line follow.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await session.moveTo({ x: 756, y: 400 });

    session.startRecording();
    await session.hold(1200);

    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    await session.click(drawer.getByText("Lines", { exact: true }).first(), {
      text: true,
    });
    const row = drawer.locator(".pluto-list__item").first();
    await session.waitFor(row);
    await session.hold(600);

    // No zoom on any of these: the plot must stay in frame so each change
    // shows up live.
    await session.click(row.locator("input").first(), { zoom: false });
    await session.press("ControlOrMeta+a");
    await session.type("Chamber pressure");
    await session.press("Enter");
    await session.hold(1000);

    await session.click(row.locator("input").nth(1), { zoom: false });
    await session.press("ControlOrMeta+a");
    await session.type("4");
    await session.press("Enter");
    await session.hold(1000);

    await session.click(row.locator(".pluto-color-swatch").first(), { zoom: false });
    const hex = page.locator(".pluto-color-picker input").first();
    await session.waitFor(hex);
    await session.hold(400);
    await session.click(hex, { zoom: false });
    await session.press("ControlOrMeta+a");
    await session.type("22C55E");
    await session.press("Enter");
    await session.hold(800);
    await session.press("Escape");
    await session.hold(2600);
  } finally {
    await fixture.stop();
  }
};
