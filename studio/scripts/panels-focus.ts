// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { createStand, mosaicTab, splitRight } from "./common/panels";

/**
 * Docs `console/panels/focus`: with a line plot and a table side by side, right-click
 * the table tab and select Focus, so the table fills the panel. Exit focus from the
 * same menu returns to both.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const stand = await createStand(session);
  try {
    const { page } = session;
    await splitRight(session, stand.table);
    await session.moveTo(capture.PARK);
    await session.settle(500);

    session.startRecording();
    await session.hold(1800);

    const leaves = page.locator(".console-mosaic .pluto-mosaic__leaf:visible");
    await capture.contextMenu(session, mosaicTab(session, stand.table), /^Focus/);
    session.endZoom();
    await session.waitFor(page.locator(".pluto-panel-mosaic__overlaid-leaf").first());
    await session.hold(2000);

    await capture.contextMenu(session, mosaicTab(session, stand.table), /^Exit focus/);
    session.endZoom();
    await session.waitForHidden(
      page.locator(".pluto-panel-mosaic__overlaid-leaf").first(),
    );
    await session.waitFor(leaves.nth(1));
    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await stand.stop();
  }
};
