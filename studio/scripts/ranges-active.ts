// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

const RANGE = "Hotfire 09";
const OTHER = "Coldflow 3";
const CHANNEL = "tank_pressure";

/**
 * Docs `console/ranges/active`: in the ranges toolbar, right-click a saved range and
 * select "Set as active range", then create a line plot. The new plot opens on the
 * active range and draws its data.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.createStaticTelemetry({ channels: [CHANNEL] });
  // Covers the static block, which ends 5 seconds before now.
  const now = TimeStamp.now();
  await fixtures.createRanges([
    {
      name: RANGE,
      timeRange: new TimeRange(
        now.sub(TimeSpan.seconds(130)),
        now.sub(TimeSpan.seconds(3)),
      ),
    },
    OTHER,
  ]);
  const { page } = session;
  await capture.login(session);

  // Opening a range from the palette adds it to the toolbar and makes it active, so
  // the other range opens last and holds the selection at the start of the shot.
  await capture.searchPalette(session, RANGE);
  await capture.searchPalette(session, OTHER);
  await capture.clearPanel(session);
  // V toggles the toolbar, so it starts closed.
  await capture.hideBottomToolbar(session);
  await capture.openToolbar(session, "range");
  const item = page
    .locator(".console-range-list-item")
    .filter({ hasText: RANGE })
    .first();
  await session.waitFor(item);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.contextMenu(session, item, "Set as active range");
  await session.hold(1200);
  session.endZoom();

  await capture.clickPanelCreate(session);
  await session.hold(800);
  await capture.createComponent(session, "Line plot");
  await session.waitFor(page.locator(".pluto-line-plot").first());
  await session.hold(1000);

  await session.press("v");
  const drawer = page.locator(".console-nav__drawer.lyra--location-bottom").first();
  const range = drawer.locator(".lyra-tag").filter({ hasText: RANGE }).first();
  await session.waitFor(range);
  await session.hold(600);
  // A region's fit caps the zoom, and the drawer spans the window, so zoom on a point
  // over its left side, where the range and channel fields are.
  const box = await drawer.boundingBox();
  if (box == null) throw new Error("line plot toolbar is not on screen");
  await session.zoom({ x: box.x + 400, y: box.y + box.height / 2 }, 1.6);
  // Rest on the X axis's range, which the plot took from the active range.
  await session.moveTo(range, { text: true });
  await session.hold(1200);

  await capture.addChannels(session, "Y1", [CHANNEL], { search: CHANNEL });
  session.endZoom();
  await session.hold(600);

  // Off the plot, so no crosshair covers the new line.
  await session.moveTo({ x: 950, y: 24 });
  await session.hold(3000);
};
