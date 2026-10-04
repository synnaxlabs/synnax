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
  const bind = async (section: string, channel: string): Promise<void> => {
    const item = properties
      .locator(".pluto-form-section")
      .filter({
        has: page
          .locator(".pluto-form-section__header")
          .getByText(section, { exact: true }),
      })
      .locator(".pluto-input__item")
      .filter({ hasText: "Channel" })
      .first();
    await session.click(item.getByText("Select channel", { exact: true }).first(), {
      zoom: false,
    });
    const search = page.getByPlaceholder("Search channels...").first();
    await session.waitFor(search);
    await search.fill(channel);
    const option = page
      .locator(".pluto-list__item")
      .filter({ has: page.getByText(channel, { exact: true }) })
      .first();
    await session.waitFor(option);
    await session.click(option, { zoom: false });
    await session.settle(400);
  };
  await bind("State", valve.state);
  await bind("Command", valve.command);

  await capture.deselect(session, { x: 100, y: 290 });
  if (authority != null) {
    await capture.selectToolbarTab(session, "Control");
    const input = page
      .locator(".console-schematic__control .pluto-input__item")
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

/** controlButton returns the schematic's acquire/release control toggle. */
export const controlButton = (session: capture.CaptureSession): Locator =>
  session.page.locator(".console-controls button:has(svg.pluto-icon--circle)").first();

/**
 * frameLegend opens an authored zoom holding both the control legend and the valve,
 * which sit far apart. Returns a point beside the valve to park the cursor on, so it
 * covers neither.
 */
export const frameLegend = async (
  session: capture.CaptureSession,
  valve: Locator,
): Promise<{ x: number; y: number }> => {
  const { page } = session;
  const [legend, body] = await Promise.all([
    page.locator(".pluto-legend-entry").first().boundingBox(),
    valve.boundingBox(),
  ]);
  const view = page.viewportSize();
  if (legend == null || body == null || view == null)
    throw new Error("legend or valve is hidden");
  const left = Math.min(legend.x, body.x);
  const top = Math.min(legend.y, body.y);
  const right = Math.max(legend.x + legend.width, body.x + body.width);
  const bottom = Math.max(legend.y + legend.height, body.y + body.height);
  await session.zoom(
    { x: (left + right) / 2, y: (top + bottom) / 2 },
    Math.min(
      1.6,
      view.width / (right - left + 192),
      view.height / (bottom - top + 192),
    ),
  );
  return { x: body.x + body.width / 2 + 180, y: body.y + body.height + 120 };
};
