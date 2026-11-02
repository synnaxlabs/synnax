// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { cells, recordStill, resize } from "./common/table";

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
    const addRow = page.locator(".pluto-table-frame__add-row .lyra-btn").last();
    const addColumn = page.locator(".pluto-table-frame__add-col .lyra-btn").last();
    await session.click(addRow, { zoom: false });
    await session.click(addRow, { zoom: false });
    await session.click(addColumn, { zoom: false });
    await session.settle(300);

    for (const column of ["A", "B", "C"])
      await resize(session, `Resize column ${column}`, { x: 100, y: 0 });
    for (const row of ["1", "2", "3", "4"])
      await resize(session, `Resize row ${row}`, { x: 0, y: 12 });

    const { setText, setValue } = cells(session, COLUMNS);

    await setText(0, 0, "Location");
    await setText(0, 1, "Pressure");
    await setText(0, 2, "Temperature");
    const names = ["Tank", "Line", "Pump"];
    for (const [i, source] of sources.entries()) {
      await setText(i + 1, 0, names[i]);
      await setValue(i + 1, 1, source.channels[0]);
      await setValue(i + 1, 2, source.channels[1]);
    }

    await recordStill(session);
  } finally {
    await Promise.all(sources.map(async (s) => await s.stop()));
  }
};
