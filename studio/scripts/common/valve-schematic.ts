// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, type fixtures } from "@/index";

export interface ValveSchematicOptions {
  /** Control authority the schematic acquires with. Unset keeps the Console default. */
  authority?: number;
}

/**
 * createValveSchematic logs in and stages a schematic holding one valve bound to the
 * fixture's state and command channels, centered on a canvas with the toolbar closed.
 * It parks the cursor inside the schematic, where the floating controls show, and
 * returns the valve. Run before recording.
 */
export const createValveSchematic = async (
  session: capture.CaptureSession,
  valve: fixtures.ValveFixture,
  { authority }: ValveSchematicOptions = {},
): Promise<Locator> => {
  const { page } = session;
  await capture.login(session);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  // "Generic" is the catalog name of the plain valve; "Valve" is its label.
  const symbol = await capture.place(session, "Generic", { x: 500, y: 200 });

  const properties = page.locator(".console-schematic__properties").first();
  await session.click(
    properties.locator("[role='tab']").filter({ hasText: "Control" }).first(),
    { text: true, zoom: false },
  );
  await capture.bindChannel(session, "State", valve.state);
  await capture.bindChannel(session, "Command", valve.command);

  await capture.deselect(session, { x: 100, y: 290 });
  if (authority != null) {
    await capture.selectToolbarTab(session, "Control");
    const input = page
      .locator(".console-schematic__control .lyra-input__item")
      .filter({ hasText: "Control authority" })
      .locator("input")
      .first();
    await session.waitFor(input);
    await input.fill(String(authority));
    await input.press("Enter");
    await session.settle(300);
  }
  // Control mode leaves the toolbar on a "not editable" placeholder, so the drawer
  // closes before recording.
  await session.click(page.locator(".console-main-nav__item").last(), { zoom: false });
  await session.settle(600);
  // The canvas only reaches its recorded size once the drawer closes.
  const pane = await capture.canvas(page).boundingBox();
  if (pane == null) throw new Error("schematic canvas is not on screen");
  await session.drag(
    symbol.locator(".pluto-drag-handle").first(),
    { x: pane.x + pane.width / 2, y: pane.y + pane.height / 2 },
    { zoom: false },
  );
  await session.settle(400);
  // The drag selects the valve.
  await capture.deselect(session, { x: 100, y: 290 });
  await session.settle(300);
  await session.moveTo({ x: 300, y: 560 });
  return symbol;
};

/**
 * frameLegend opens an authored zoom holding both the control legend and the valve,
 * which sit far apart. Returns a point beside the valve to park the cursor on, so it
 * covers neither.
 */
export const frameLegend = async (
  session: capture.CaptureSession,
  valve: Locator,
): Promise<{ x: number; y: number }> => {
  await session.zoom([session.page.locator(".pluto-legend-entry").first(), valve], 1.6);
  const body = await valve.boundingBox();
  if (body == null) throw new Error("valve is hidden");
  return { x: body.x + body.width / 2 + 180, y: body.y + body.height + 120 };
};
