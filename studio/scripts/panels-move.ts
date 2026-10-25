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
  createPanel,
  createStand,
  mosaicTab,
  NEW_PANEL,
  pill,
  selectedPill,
  selectPanel,
  STAND,
} from "./common/panels";

/**
 * Docs `console/panels/move`: drag the table tab onto the New panel name, hold until
 * that panel opens, and drop the tab in it.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const stand = await createStand(session);
  try {
    const { page } = session;
    await createPanel(session, NEW_PANEL);
    await selectPanel(session, STAND);
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    // The panel opens after the tab rests on its name for 450 ms.
    const mosaic = page.locator(".console-mosaic").first();
    await session.drag(
      mosaicTab(session, stand.table),
      [pill(session, NEW_PANEL), mosaic],
      { zoom: false, pauseMs: 1000 },
    );
    await session.waitFor(selectedPill(session, NEW_PANEL));
    await session.waitFor(mosaicTab(session, stand.table));
    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await stand.stop();
  }
};
