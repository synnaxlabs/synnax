// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { cells, drawer, recordStill, resize } from "./common/table";

const COLUMNS = 2;
const WARNING = 70;
const ERROR = 85;

/**
 * Still for docs image `console/tables/redline`: three live pressure values whose
 * redline has a warning band and a higher error band. One value sits below both
 * bands, one in the warning band, and one in the error band. The last captured frame
 * is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // Each source swings a little around a base inside one band, so the colors hold.
  const sources = await Promise.all(
    [
      { prefix: "tank", base: 55 },
      { prefix: "feed", base: 77 },
      { prefix: "pump", base: 95 },
    ].map(async (opts) => await fixtures.sineTelemetry({ ...opts, swing: 3 })),
  );
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Table");
    await session.waitFor(page.locator(".console-table").first());
    const addRow = page.locator(".pluto-table-frame__add-row .lyra-btn").last();
    await session.click(addRow, { zoom: false });
    await session.click(addRow, { zoom: false });
    await session.settle(300);

    for (const column of ["A", "B"])
      await resize(session, `Resize column ${column}`, { x: 120, y: 0 });
    for (const row of ["1", "2", "3", "4"])
      await resize(session, `Resize row ${row}`, { x: 0, y: 12 });

    const { setText, setValue } = cells(session, COLUMNS);
    const toolbar = drawer(page);
    // The first added band takes the warning color and the second the error color.
    // Bands sort by threshold, so the error band moves up before the warning band.
    const setRedline = async () => {
      await session.click(toolbar.getByText("Redline", { exact: true }).first(), {
        text: true,
        zoom: false,
      });
      const add = toolbar.getByRole("button", { name: "Add band" }).first();
      await session.waitFor(add);
      await session.click(add, { zoom: false });
      await session.click(add, { zoom: false });
      const threshold = (index: number) =>
        toolbar
          .locator(".pluto-redline-form__bands .lyra-list__item")
          .nth(index)
          .locator("input[type='text']")
          .first();
      for (const [index, value] of [
        [1, ERROR],
        [0, WARNING],
      ]) {
        const input = threshold(index);
        await session.waitFor(input);
        await input.fill(String(value));
        await input.press("Enter");
        await session.settle(300);
      }
    };

    await setText(0, 0, "Location");
    await setText(0, 1, "Pressure");
    const names = ["Tank", "Feed", "Pump"];
    for (const [i, source] of sources.entries()) {
      await setText(i + 1, 0, names[i]);
      await setValue(i + 1, 1, source.channels[0]);
      await setRedline();
    }

    await recordStill(session);
  } finally {
    await Promise.all(sources.map(async (s) => await s.stop()));
  }
};
