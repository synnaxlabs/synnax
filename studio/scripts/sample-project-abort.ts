// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { button, openSample, symbol, watchTank } from "./common/sample";

export const desktop = true;

/**
 * Docs `get-started/sample-project/abort`: click Start, then Abort before the hold
 * ends. The Press sequence closes the press valve and opens the vent valve, and the
 * pressure falls.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openSample(session);
  const tank = await watchTank();
  try {
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);
    await session.zoom([symbol(page, "Start"), symbol(page, "Abort")], 1.6);
    await session.hold(800);
    await session.click(button(page, "Start"));
    await session.waitFor(() => tank.press === 1);
    // After the hold, the sequence vents by itself, and the shot would show no abort.
    if (tank.vent === 1) throw new Error("the test vented before the Abort click");
    await session.click(button(page, "Abort"));
    session.endZoom();
    await session.moveTo(capture.PARK);
    await session.waitFor(() => tank.press === 0 && tank.vent === 1);
    await session.waitFor(() => tank.pressure < 20, 3600);
    await session.hold(2500);
  } finally {
    await tank.stop();
  }
};
