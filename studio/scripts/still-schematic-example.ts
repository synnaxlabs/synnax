// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

/**
 * Still for docs image `console/schematics/example`: a tank, a pump, and three valves
 * on one line, with live pressure and temperature values and editing off. The last
 * captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Schematic");
    await session.waitFor(page.locator(".pluto-diagram").first());
    // Measure the canvas with the toolbar open, since the toolbar covers its bottom.
    await capture.selectToolbarTab(session, "Symbols");
    const pane = await capture.canvas(page).boundingBox();
    if (pane == null) throw new Error("schematic canvas is not on screen");
    const at = (x: number, y: number): capture.CanvasPosition => ({
      x: pane.width * x,
      y: pane.height * y,
    });

    // New symbols drop at the canvas center, so the two closest to it go last.
    const tank = await capture.place(session, "Tank", at(0.14, 0.62));
    const ball = await capture.place(session, "Ball", at(0.3, 0.68));
    const solenoid = await capture.place(session, "Solenoid", at(0.78, 0.68));
    // High enough that the box clears the tank's label.
    const pressure = await capture.place(session, "Value", at(0.14, 0.1));
    const temperature = await capture.place(session, "Value", at(0.62, 0.42));
    const pump = await capture.place(session, "Pump", at(0.46, 0.68));
    const check = await capture.place(session, "Check", at(0.62, 0.68));

    const drawer = page.locator(".console-nav__drawer.pluto--location-bottom").first();
    const field = (label: string): Locator =>
      drawer
        .locator(".pluto-input__item")
        .filter({ hasText: new RegExp(`^${label}`) })
        .locator("input")
        .first();
    const bind = async (
      value: Locator,
      channel: string,
      label: string,
      units: string,
    ): Promise<void> => {
      await session.click(value, { zoom: false });
      await capture.selectToolbarTab(session, "Properties");
      await session.click(
        drawer.locator("[role='tab']").filter({ hasText: "Style" }).first(),
        { text: true, zoom: false },
      );
      for (const [name, text] of [
        ["Label", label],
        ["Units", units],
      ]) {
        const input = field(name);
        await session.waitFor(input);
        await input.fill(text);
        await input.press("Enter");
        await session.settle(300);
      }
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
      await session.settle(400);
    };
    await bind(pressure, fixture.channels[0], "Tank pressure", "psi");
    await bind(temperature, fixture.channels[1], "Line temperature", "°C");

    const line = [tank, ball, pump, check, solenoid];
    for (let i = 0; i < line.length - 1; i++) {
      await session.drag(
        capture.handle(line[i], "right"),
        capture.handle(line[i + 1], "left"),
        { zoom: false },
      );
      // A drop onto a handle leaves the new edge selected.
      await page.keyboard.press("Escape");
    }
    await capture.deselect(session, at(0.5, 0.06));

    await session.click(capture.control(page, "edit-off"), { zoom: false });
    await capture.hideBottomToolbar(session);
    await session.settle(600);
    await session.click(capture.control(page, "expand"), { zoom: false });
    await session.settleWall(2000);
    await session.settle(500);
    // Off the panel: the controls fade and no symbol shows its hover state.
    await session.moveTo({ x: 900, y: 24 });

    session.startRecording();
    await session.hold(2500);
  } finally {
    await fixture.stop();
  }
};
