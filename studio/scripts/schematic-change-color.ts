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
 * Docs `console/schematics/change-color`: drag a selection box around three
 * symbols, open the selection's color picker, and enter a new color; all three
 * recolor together.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const project = `Docs Videos ${Date.now().toString(36)}`;
  await capture.login(session, { username: "synnax", password: "seldon" }, project);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  const tank = await capture.place(session, "Tank", { x: 450, y: 300 });
  const valve = await capture.place(session, "Gate", { x: 780, y: 300 });
  const pump = await capture.place(session, "Pump", { x: 1060, y: 300 });
  await capture.deselect(session, { x: 200, y: 600 });
  await session.moveTo({ x: 300, y: 620 });

  session.startRecording();
  await session.hold(1200);

  await capture.selectSymbols(session, [tank, valve, pump]);
  await session.hold(1000);

  // No zoom on any of these: the symbols must stay in frame so the recolor
  // shows up live.
  const swatch = page
    .locator(".console-schematic__properties-multi .pluto-color-swatch")
    .first();
  await session.waitFor(swatch);
  await session.click(swatch, { zoom: false });
  const hex = page.locator(".pluto-color-picker input").first();
  await session.waitFor(hex);
  await session.hold(600);

  await session.click(hex, { zoom: false });
  await session.press("ControlOrMeta+a");
  await session.type("EF4444");
  await session.press("Enter");
  await session.hold(1000);

  await session.press("Escape");
  await session.hold(2600);
};
