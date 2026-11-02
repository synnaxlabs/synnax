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
  absoluteChip,
  button,
  chip,
  lostChip,
  openSample,
  symbol,
  toggle,
  watchTank,
} from "./common/sample";

export const desktop = true;

/**
 * Docs `get-started/sample-project/take-control`: take the vent valve with its control
 * chip and open it, then click Start. The Press sequence cannot close the vent valve,
 * so the pressure levels off. Close the vent valve and give it back with the chip, and
 * the test goes on.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openSample(session);
  const tank = await watchTank();
  try {
    const vent = symbol(page, "Vent valve");
    const ventBox = await vent.boundingBox();
    const readout = await symbol(page, "Tank pressure").boundingBox();
    if (ventBox == null || readout == null) throw new Error("the schematic is hidden");
    // Beside the pressure readout, where no symbol shows its channel tooltip over it.
    const rest = {
      x: ventBox.x + ventBox.width / 2,
      y: readout.y + readout.height / 2,
    };
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);
    await session.zoom(
      [
        vent,
        symbol(page, "Tank pressure"),
        symbol(page, "Start"),
        symbol(page, "Abort"),
      ],
      1.5,
    );
    await session.hold(800);
    await session.click(chip(vent));
    await session.waitFor(absoluteChip(vent));
    await session.hold(600);
    await session.click(toggle(vent));
    await session.moveTo(rest);
    await session.waitFor(() => tank.vent === 1);
    await session.hold(800);
    await session.click(button(page, "Start"));
    await session.moveTo(rest);
    await session.waitFor(() => tank.press === 1);
    // With both valves open, the tank settles near 142 psi.
    await session.waitFor(() => tank.pressure > 125, 3600);
    await session.hold(1500);
    await session.click(toggle(vent));
    await session.moveTo(rest);
    await session.waitFor(() => tank.vent === 0);
    await session.hold(600);
    await session.click(chip(vent));
    await session.waitFor(lostChip(vent));
    session.endZoom();
    await session.moveTo(capture.PARK);
    // The test goes on: the sequence closes the press valve to hold at 300 psi.
    await session.waitFor(() => tank.press === 0, 3600);
    await session.hold(2500);
  } finally {
    await tank.stop();
  }
};
