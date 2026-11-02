// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { frameLegend } from "./valve-schematic";

/**
 * arduinoSwitchScript returns a script for the custom devices tutorial: bind a
 * schematic switch to the Arduino's command channel and to `state`, take control, and
 * click the switch on.
 */
export const arduinoSwitchScript =
  (state: (fixture: fixtures.ArduinoFixture) => string) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const fixture = await fixtures.arduino();
    try {
      const { page } = session;
      await capture.login(session);

      await capture.createComponent(session, "Schematic");
      await session.waitFor(page.locator(".pluto-diagram").first());
      const symbol = await capture.place(session, "Switch", { x: 520, y: 170 });
      await capture.deselect(session, { x: 100, y: 290 });
      await session.moveTo({ x: 800, y: 620 });

      session.startRecording();
      await session.hold(1800);

      await session.click(symbol.locator(".pluto-drag-handle").first(), {
        zoom: false,
      });
      await session.hold(600);
      const properties = page.locator(".console-schematic__properties").first();
      await session.waitFor(properties);
      // The drawer spans the window, so a zoom on it renders at 1x. Zoom on its left
      // half, where the State and Command fields sit.
      const drawer = await properties.boundingBox();
      if (drawer == null) throw new Error("properties drawer is hidden");
      await session.zoom(
        { x: drawer.x + drawer.width * 0.42, y: drawer.y + drawer.height / 2 },
        1.5,
      );
      await session.click(
        properties.locator("[role='tab']").filter({ hasText: "Control" }).first(),
        { text: true, zoom: false },
      );
      await session.hold(500);
      const bind = async (section: string, channel: string): Promise<void> => {
        const item = capture.channelField(properties, section);
        await session.click(item.getByText("Select channel", { exact: true }).first(), {
          text: true,
          zoom: false,
        });
        const search = page.getByPlaceholder("Search channels...").first();
        await session.waitFor(search);
        await session.hold(400);
        session.setSpeed(1.5);
        await session.type(channel);
        session.setSpeed(1);
        const option = page
          .locator(".lyra-list__item")
          .filter({ has: page.getByText(channel, { exact: true }) })
          .first();
        await session.waitFor(option);
        await session.hold(300);
        await session.click(option.getByText(channel, { exact: true }).first(), {
          text: true,
          zoom: false,
        });
        await session.hold(700);
      };
      await bind("Command", fixture.command);
      await bind("State", state(fixture));
      session.endZoom();

      await capture.deselect(session, { x: 100, y: 290 });
      await session.click(page.locator(".console-main-nav__item").last(), {
        zoom: false,
      });
      await session.settle(600);
      await session.moveTo({ x: 800, y: 620 });
      await session.hold(600);

      await session.click(capture.control(page, "circle"), { zoom: false });
      await session.waitFor(page.locator(".pluto-legend-entry").first());
      await session.hold(1000);

      const aside = await frameLegend(session, symbol);
      await session.moveTo(aside);
      await session.hold(900);
      await session.click(symbol.locator(".lyra-input__switch-indicator").first(), {
        zoom: false,
      });
      // The command reaches the fixture, and its echo comes back, on wall time.
      await session.settleWall(800);
      await session.moveTo(aside);
      await session.hold(1400);
      session.endZoom();
      await session.hold(2800);
    } finally {
      await fixture.stop();
    }
  };
