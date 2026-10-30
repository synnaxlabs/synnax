// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator, type Page } from "playwright";

import { capture } from "@/index";

/** drawer returns the bottom toolbar, which edits the selected cell. */
export const drawer = (page: Page): Locator =>
  page.locator(".console-nav__drawer.lyra--location-bottom").first();

/** resize drags the table's resize handle named `handle` by `by`, off camera. */
export const resize = async (
  session: capture.CaptureSession,
  handle: string,
  by: { x: number; y: number },
): Promise<void> => {
  const button = session.page
    .getByRole("button", { name: handle, exact: true })
    .first();
  const box = await button.boundingBox();
  if (box == null) throw new Error(`${handle} is not on screen`);
  const from = { x: box.x + box.width / 2, y: box.y + box.height / 2 };
  await session.drag(from, { x: from.x + by.x, y: from.y + by.y }, { zoom: false });
};

/** Cells edits the cells of a table off camera. */
export interface Cells {
  /** setText makes the cell a text cell that reads `text`. */
  setText: (row: number, column: number, text: string) => Promise<void>;
  /** setValue makes the cell a value cell that shows the live value of `channel`. */
  setValue: (row: number, column: number, channel: string) => Promise<void>;
}

/** cells returns the editors of a table `columns` cells wide. */
export const cells = (session: capture.CaptureSession, columns: number): Cells => {
  const { page } = session;
  const toolbar = drawer(page);
  // Clicking a cell does not reopen a closed toolbar, and V toggles it.
  const showToolbar = async (): Promise<void> => {
    await session.settle(200);
    const change = toolbar.locator("[aria-label='Change cell type']").first();
    if (!(await change.isVisible())) await session.press("v");
    await session.waitFor(change);
  };
  const cell = (row: number, column: number): Locator =>
    page.locator(".pluto-table__cell").nth(row * columns + column);
  return {
    setText: async (row, column, text) => {
      await session.click(cell(row, column), { zoom: false });
      await showToolbar();
      const input = toolbar.getByRole("textbox", { name: "Text", exact: true }).first();
      await session.waitFor(input);
      await input.fill(text);
      await input.press("Enter");
      await session.settle(300);
    },
    setValue: async (row, column, channel) => {
      await session.click(cell(row, column), { zoom: false });
      await showToolbar();
      await session.click(toolbar.locator("[aria-label='Change cell type']").first(), {
        zoom: false,
      });
      const option = page
        .locator(".pluto-table-cell-change-variant__dialog [role='option']")
        .filter({ hasText: "Value" })
        .first();
      await session.waitFor(option);
      await session.click(option, { text: true, zoom: false });
      const telemetry = toolbar.getByText("Telemetry", { exact: true }).first();
      await session.waitFor(telemetry);
      await session.click(telemetry, { text: true, zoom: false });
      await session.click(
        toolbar.getByText("Select channel", { exact: true }).first(),
        {
          zoom: false,
        },
      );
      await capture.pickChannel(session, channel);
      await session.settle(300);
    },
  };
};

/**
 * recordStill turns off editing and the indicators, aligns the table to the top left,
 * and records it zoomed in. The last captured frame is the still.
 */
export const recordStill = async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await session.click(capture.control(page, "edit-off"), { zoom: false });
  await session.click(capture.control(page, "invisible"), { zoom: false });
  await session.click(capture.control(page, "align-box-top-left"), { zoom: false });
  await capture.hideBottomToolbar(session);
  await session.settleWall(2000);
  await session.settle(500);
  // Off the panel, so the controls fade.
  await session.moveTo(capture.PARK);

  session.startRecording();
  await session.zoom(page.locator(".pluto-table").first(), 1.6);
  await session.hold(2500);
};
