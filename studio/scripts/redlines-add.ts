// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { cells, drawer } from "./common/table";

const COLUMNS = 2;

/**
 * Docs `reference/alarms/redlines` (Add a redline): select a value cell of a table,
 * press V, open the Redline tab, click Add band twice, and set the thresholds. The live
 * value takes the color of its band.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry({ prefix: "tank", base: 80, swing: 4 });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.createComponent(session, "Table");
    await session.waitFor(page.locator(".console-table").first());
    const { setText, setValue } = cells(session, COLUMNS);
    await setText(0, 0, "Location");
    await setText(0, 1, "Pressure");
    await setText(1, 0, "Tank");
    await setValue(1, 1, fixture.channels[0]);
    // A selected cell holds the toolbar open.
    await session.click({ x: 900, y: 600 }, { zoom: false });
    await capture.hideBottomToolbar(session);
    await session.settleWall(2000);
    await session.moveTo({ x: 900, y: 600 });

    session.startRecording();
    await session.hold(1800);

    const value = page.locator(".pluto-table__cell").nth(3);
    await session.click(value, { zoom: false });
    await session.hold(600);
    const toolbar = drawer(page);
    const tab = toolbar.getByText("Redline", { exact: true }).first();
    if (!(await tab.isVisible())) await session.press("v");
    await session.waitFor(tab);
    await session.hold(800);
    await capture.zoomCenter(session, toolbar, 1.4);
    await session.click(tab, { text: true });
    const add = toolbar.getByRole("button", { name: "Add band" }).first();
    await session.waitFor(add);
    await session.hold(800);

    await session.click(add);
    await session.hold(600);
    await session.click(add);
    await session.hold(600);
    const threshold = (index: number) =>
      toolbar
        .locator(".pluto-redline-form__bands .lyra-list__item")
        .nth(index)
        .locator("input[type='text']")
        .first();
    for (const [index, level] of [
      [1, "90"],
      [0, "70"],
    ] as const) {
      await session.click(threshold(index));
      await session.press("ControlOrMeta+a");
      await session.type(level);
      await session.press("Enter");
      await session.hold(500);
    }
    session.endZoom();
    await session.moveTo({ x: 900, y: 300 });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
