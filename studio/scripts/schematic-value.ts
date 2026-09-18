// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

/**
 * Docs `console/schematics/value`: select a value symbol, pick its input
 * channel in the Telemetry tab, and raise the precision; the symbol starts
 * showing the channel's live value.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.createComponent(session, "Schematic");
    await session.waitFor(page.locator(".pluto-diagram").first());
    const value = await capture.place(session, "Value", { x: 720, y: 300 });
    await capture.deselect(session, { x: 200, y: 600 });
    await session.moveTo({ x: 300, y: 620 });

    session.startRecording();
    await session.hold(1200);

    // No zoom anywhere: the symbol must stay in frame so the live value shows
    // up the moment the channel is bound.
    await session.click(value, { zoom: false });
    await session.hold(800);

    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    await session.click(drawer.getByText("Telemetry", { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    await session.hold(700);

    await session.click(drawer.getByText("Select channel", { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    const search = page.locator("input[placeholder*='Search']").first();
    await session.waitFor(search);
    await session.hold(600);
    session.setSpeed(1.5);
    await session.type(fixture.channels[0]);
    session.setSpeed(1);
    const item = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: fixture.channels[0] })
      .first();
    await session.waitFor(item);
    await session.hold(400);
    await session.click(item.getByText(fixture.channels[0], { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    await session.hold(1200);

    const precision = drawer
      .getByText("Precision", { exact: true })
      .locator("..")
      .locator("input")
      .first();
    await session.click(precision, { zoom: false });
    await session.press("ControlOrMeta+a");
    await session.type("3");
    await session.press("Enter");
    await session.hold(1000);

    await session.moveTo({ x: 300, y: 620 });
    // Live telemetry arrives on wall time, so give the stream real time to
    // push samples before the closing hold.
    await session.settleWall(1500);
    await session.hold(2600);
  } finally {
    await fixture.stop();
  }
};
