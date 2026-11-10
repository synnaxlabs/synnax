// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import {
  addPanel,
  createStand,
  mosaicTab,
  NEW_PANEL,
  pill,
  STAND,
  zoomSelector,
} from "./common/panels";

/**
 * Docs `console/panels/create`: click the add button at the right of the panel
 * selector to open New panel, click Test stand to switch back, then drag Test stand
 * after New panel.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const stand = await createStand(session);
  try {
    const { page } = session;
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    await zoomSelector(session);
    await session.click(addPanel(session), { zoom: false });
    await session.waitFor(pill(session, NEW_PANEL));
    await session.waitFor(page.getByText("No components open").first());
    await session.hold(1400);

    await session.click(pill(session, STAND), { text: true, zoom: false });
    await session.waitFor(mosaicTab(session, stand.plot));
    await session.hold(1200);

    const target = await pill(session, NEW_PANEL).boundingBox();
    if (target == null) throw new Error("the New panel pill is not on screen");
    await session.drag(
      pill(session, STAND),
      { x: target.x + target.width * 0.8, y: target.y + target.height / 2 },
      { zoom: false },
    );
    session.endZoom();
    await session.waitFor(
      page.locator(".console-panel-selector .lyra-tabs__tab").last().filter({
        hasText: STAND,
      }),
    );
    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await stand.stop();
  }
};
