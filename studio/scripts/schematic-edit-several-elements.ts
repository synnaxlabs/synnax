// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

/**
 * Docs `console/schematics/edit-several-elements`: drag a selection box around three
 * symbols and set their shared stroke color. All three recolor together, and the
 * Selection control shows the new color.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  // New symbols drop at the canvas center, so only the last one may sit there.
  const gate = await capture.place(session, "Gate", { x: 200, y: 160 });
  const pump = await capture.place(session, "Pump", { x: 830, y: 160 });
  const light = await capture.place(session, "Light", { x: 515, y: 160 });
  await capture.deselect(session, { x: 100, y: 290 });
  await session.moveTo({ x: 150, y: 350 });

  session.startRecording();
  await session.hold(500);

  await capture.selectSymbols(session, [gate, light, pump]);
  await session.hold(1000);

  const swatch = page.locator(".console-multiedit-colors .pluto-color-swatch").first();
  await session.waitFor(swatch);
  await session.click(swatch, { zoom: false });
  const hex = page.locator(".pluto-color-picker .pluto-color-hex-text input");
  await session.waitFor(hex);
  await session.zoom(page.locator(".pluto-color-picker").first());
  await session.hold(600);

  await session.click(hex, { zoom: false });
  await session.press("ControlOrMeta+a");
  await session.type("EF4444");
  // Leave the picker before Enter so the recolor lands in a wide frame.
  session.endZoom();
  await session.hold(300);
  await session.press("Enter");
  await session.hold(1000);

  // The empty foot of the Colors group is the nearest outside click that closes the
  // picker and changes nothing.
  const colors = await page.locator(".console-multiedit-colors").boundingBox();
  if (colors == null) throw new Error("colors group is not on screen");
  await session.click(
    { x: colors.x + colors.width / 2, y: colors.y + colors.height - 8 },
    { zoom: false },
  );
  await session.moveTo({ x: 150, y: 350 });
  await session.hold(1000);
};
