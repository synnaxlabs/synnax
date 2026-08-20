// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const Y1 = "label:has-text('Y1')";

/**
 * plotY1 selects one channel onto the focused plot's Y1 axis. The shared helper
 * lets the camera punch into the dropdown, which throws this shot off its
 * subject: every click here holds the panel in frame instead.
 */
const plotY1 = async (
  session: capture.CaptureSession,
  channel: string,
): Promise<void> => {
  const { page } = session;
  const trigger = page
    .locator("label")
    .filter({ hasText: "Y1" })
    .locator("..")
    .locator(".pluto-dialog__trigger")
    .first();
  await session.click(trigger.getByText(/Select channel/).first(), { zoom: false });
  const input = page.locator("input[placeholder*='Search']");
  await session.waitFor(input);
  await session.hold(500);
  await session.type("demo");
  const item = page
    .locator(".pluto-list__item:not(.pluto-tree__item)")
    .filter({ hasText: channel })
    .first();
  await session.waitFor(item);
  await session.click(item.getByText(channel, { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await session.press("Escape");
};

/**
 * Docs `console/ui-overview/multiple-tabs`: with one streaming line plot open,
 * create a second tab, plot a channel on it, then drag its tab to the right edge
 * of the mosaic to split the panel into two visualizations side by side.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideVisualizationToolbar(session);
    // Live telemetry buffers on wall time: fill the rolling window before the
    // first frame so the plot never records as an empty grid.
    await session.settleWall(24000);
    await session.settle(1000);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1200);

    const tabs = page.locator(".console-mosaic .pluto-tabs__tab");
    await capture.clickPanelCreate(session);
    await session.hold(700);
    await capture.createComponent(session, "Line plot");
    await session.waitFor(tabs.nth(1));
    await session.hold(800);

    const visualization = page.locator(".console-main-nav__item").last();
    // No zoom on either toggle: the drawer resizes the whole panel.
    await session.click(visualization, { zoom: false });
    await session.waitFor(page.locator(Y1).first());
    await plotY1(session, fixture.channels[0]);
    await session.hold(900);
    await session.click(visualization, { zoom: false });
    await session.waitForHidden(page.locator(Y1).first());
    // The rail's toggle slides down as the drawer closes, re-entering the parked
    // cursor and raising its tooltip. Step off the rail before it does.
    await session.moveTo({ x: 756, y: 300 });
    await session.hold(600);

    // Drop point is 85% across the leaf: the mosaic resolves an edge drop into a
    // split, and anything nearer the middle drops the tab back into the strip.
    const leaf = page.locator(".pluto-mosaic__leaf").first();
    const b = await leaf.boundingBox();
    if (b == null) throw new Error("mosaic leaf has no bounding box");
    await session.drag(
      tabs.nth(1),
      { x: b.x + b.width * 0.85, y: b.y + b.height * 0.5 },
      { zoom: false },
    );
    await session.waitFor(page.locator(".pluto-mosaic__leaf").nth(1));
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
