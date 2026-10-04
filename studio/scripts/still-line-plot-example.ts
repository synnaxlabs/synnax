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
 * Still for docs image `console/line-plots/example`: a line plot with three live
 * channels on Y1 and its legend. The last captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const tank = await fixtures.sineTelemetry({ prefix: "tank" });
  // A different rate keeps the second pressure out of phase with the first.
  const line = await fixtures.sineTelemetry({ prefix: "line", periodMs: 55 });
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", tank.channels, { search: "tank" });

    // fill replaces any query left over from the first pick.
    const trigger = page
      .locator("label")
      .filter({ hasText: "Y1" })
      .locator("..")
      .locator(".pluto-dialog__trigger")
      .first();
    await session.click(trigger);
    const search = page.locator("input[placeholder*='Search']").first();
    await session.waitFor(search);
    await search.fill("line");
    const item = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: line.channels[0] })
      .first();
    await session.waitFor(item);
    await session.click(item.getByText(line.channels[0], { exact: true }).first(), {
      text: true,
    });
    await session.press("Escape");
    await capture.hideBottomToolbar(session);

    // The rolling window fills on wall time.
    await session.settleWall(30000);
    await session.settle(500);
    // Off the plot, so no crosshair or readout shows.
    await session.moveTo({ x: 900, y: 24 });

    session.startRecording();
    await session.hold(2500);
  } finally {
    await Promise.all([tank.stop(), line.stop()]);
  }
};
