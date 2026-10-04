// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const COLUMNS = 3;

/**
 * Still for docs image `console/tables/thumbnail`: a header row of labels over three
 * rows of live pressure and temperature values, shown with editing off. The last
 * captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // Distinct rates keep the three rows out of phase with each other.
  const sources = await Promise.all(
    [
      { prefix: "tank", periodMs: 40 },
      { prefix: "line", periodMs: 55 },
      { prefix: "pump", periodMs: 70 },
    ].map(async (opts) => await fixtures.sineTelemetry(opts)),
  );
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Table");
    await session.waitFor(page.locator(".console-table").first());
    const addRow = page.locator(".pluto-table-frame__add-row .pluto-btn").last();
    const addColumn = page.locator(".pluto-table-frame__add-col .pluto-btn").last();
    await session.click(addRow, { zoom: false });
    await session.click(addRow, { zoom: false });
    await session.click(addColumn, { zoom: false });
    await session.settle(300);

    const resize = async (name: string, by: { x: number; y: number }) => {
      const handle = page.getByRole("button", { name, exact: true }).first();
      const box = await handle.boundingBox();
      if (box == null) throw new Error(`${name} is not on screen`);
      const from = { x: box.x + box.width / 2, y: box.y + box.height / 2 };
      await session.drag(from, { x: from.x + by.x, y: from.y + by.y }, { zoom: false });
    };
    for (const column of ["A", "B", "C"])
      await resize(`Resize column ${column}`, { x: 100, y: 0 });
    for (const row of ["1", "2", "3", "4"])
      await resize(`Resize row ${row}`, { x: 0, y: 12 });

    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    // Clicking a cell does not reopen a closed toolbar, and V toggles it.
    const showToolbar = async (): Promise<void> => {
      await session.settle(200);
      const change = drawer.locator("[aria-label='Change cell type']").first();
      if (!(await change.isVisible())) await session.press("v");
      await session.waitFor(change);
    };
    const cell = (row: number, column: number) =>
      page.locator(".pluto-table__cell").nth(row * COLUMNS + column);
    const setText = async (row: number, column: number, text: string) => {
      await session.click(cell(row, column), { zoom: false });
      await showToolbar();
      const input = drawer.getByRole("textbox", { name: "Text", exact: true }).first();
      await session.waitFor(input);
      await input.fill(text);
      await input.press("Enter");
      await session.settle(300);
    };
    const setValue = async (row: number, column: number, channel: string) => {
      await session.click(cell(row, column), { zoom: false });
      await showToolbar();
      await session.click(drawer.locator("[aria-label='Change cell type']").first(), {
        zoom: false,
      });
      const option = page
        .locator(".pluto-table-cell-change-variant__dialog [role='option']")
        .filter({ hasText: "Value" })
        .first();
      await session.waitFor(option);
      await session.click(option, { text: true, zoom: false });
      const telemetry = drawer.getByText("Telemetry", { exact: true }).first();
      await session.waitFor(telemetry);
      await session.click(telemetry, { text: true, zoom: false });
      await session.click(drawer.getByText("Select channel", { exact: true }).first(), {
        zoom: false,
      });
      const search = page.locator("input[placeholder*='Search']").first();
      await session.waitFor(search);
      await search.fill(channel);
      const item = page
        .locator(".pluto-list__item:not(.pluto-tree__item)")
        .filter({ hasText: channel })
        .first();
      await session.waitFor(item);
      await session.click(item.getByText(channel, { exact: true }).first(), {
        text: true,
        zoom: false,
      });
      await session.settle(300);
    };

    await setText(0, 0, "Location");
    await setText(0, 1, "Pressure");
    await setText(0, 2, "Temperature");
    const names = ["Tank", "Line", "Pump"];
    for (const [i, source] of sources.entries()) {
      await setText(i + 1, 0, names[i]);
      await setValue(i + 1, 1, source.channels[0]);
      await setValue(i + 1, 2, source.channels[1]);
    }

    await session.click(capture.control(page, "edit-off"), { zoom: false });
    await session.click(capture.control(page, "invisible"), { zoom: false });
    await session.click(capture.control(page, "align-box-top-left"), { zoom: false });
    await capture.hideBottomToolbar(session);
    await session.settleWall(2000);
    await session.settle(500);
    // Off the panel, so the controls fade.
    await session.moveTo({ x: 900, y: 24 });

    session.startRecording();
    await session.zoom(page.locator(".pluto-table").first(), 1.6);
    await session.hold(2500);
  } finally {
    await Promise.all(sources.map(async (s) => await s.stop()));
  }
};
