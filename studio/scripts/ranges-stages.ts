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

const RANGE = "Hotfire 10";

/**
 * Docs `console/ranges/stages`: on a range overview, click the stage of a To do range
 * and select "Start", then click it again and select "Complete". The timeline shows
 * the start, then the end and the duration.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // A start in the future makes the range To do. The page clock trails wall time,
  // so the start stays in the future on screen.
  const start = TimeStamp.now().add(TimeSpan.hours(1));
  await fixtures.createRanges([
    { name: RANGE, timeRange: new TimeRange(start, start.add(TimeSpan.minutes(15))) },
  ]);
  const { page } = session;
  await capture.login(session);

  await capture.searchPalette(session, RANGE);
  const overview = page.locator(".console-range__overview").first();
  const stage = overview.locator(".pluto-stage-button__trigger").first();
  await session.waitFor(stage.filter({ hasText: "To do" }));
  await session.moveTo({ x: 800, y: 450 });

  const transition = async (
    name: string,
    next: string,
    committed = false,
  ): Promise<void> => {
    await session.click(stage, { text: true });
    const menu = page.locator(".pluto-stage-button__menu").first();
    await session.waitFor(menu);
    await session.hold(400);
    // Hovering an item shows what it changes, so linger before the click.
    const item = menu.getByText(name, { exact: true }).first();
    await session.moveTo(item, { text: true });
    await session.hold(1200);
    if (committed) session.endZoom();
    await session.click(item, { text: true });
    await session.waitFor(stage.filter({ hasText: next }));
  };

  session.startRecording();
  await session.hold(1500);
  // The Details block holds the stage button and the timeline it changes.
  await session.zoom(overview.locator(":scope > div > *").first());

  await transition("Start", "In progress");
  await session.hold(2000);
  await transition("Complete", "Completed", true);

  // Clear of the timeline, so the end and the duration read.
  await session.moveTo({ x: 800, y: 600 });
  await session.hold(3000);
};
