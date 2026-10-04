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
 * arduinoSwitchScript returns a script for the custom devices tutorial: bind a
 * schematic switch to the Arduino's command channel and to `state`, take control,
 * and click the switch on.
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
      await session.moveTo({ x: 300, y: 560 });

      session.startRecording();
      await session.hold(1000);

      await session.click(symbol.locator(".pluto-drag-handle").first(), {
        zoom: false,
      });
      await session.hold(600);
      const properties = page.locator(".console-schematic__properties").first();
      await session.click(
        properties.locator("[role='tab']").filter({ hasText: "Control" }).first(),
        { text: true, zoom: false },
      );
      await session.hold(500);
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
          .locator(".pluto-list__item")
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

      await capture.deselect(session, { x: 100, y: 290 });
      await session.click(page.locator(".console-main-nav__item").last(), {
        zoom: false,
      });
      await session.settle(600);
      await session.moveTo({ x: 300, y: 560 });
      await session.hold(600);

      await session.click(
        page.locator(".console-controls button:has(svg.pluto-icon--circle)").first(),
        { zoom: false },
      );
      await session.waitFor(page.locator(".pluto-legend-entry").first());
      await session.hold(1000);

      // One frame holds both the control legend and the switch, which sit far apart.
      const [legend, body] = await Promise.all([
        page.locator(".pluto-legend-entry").first().boundingBox(),
        symbol.boundingBox(),
      ]);
      const view = page.viewportSize();
      if (legend == null || body == null || view == null)
        throw new Error("legend or switch is hidden");
      const left = Math.min(legend.x, body.x);
      const top = Math.min(legend.y, body.y);
      const right = Math.max(legend.x + legend.width, body.x + body.width);
      const bottom = Math.max(legend.y + legend.height, body.y + body.height);
      const aside = { x: body.x + body.width / 2 + 160, y: body.y + body.height + 100 };
      await session.moveTo(aside);
      await session.zoom(
        { x: (left + right) / 2, y: (top + bottom) / 2 },
        Math.min(
          1.7,
          view.width / (right - left + 192),
          view.height / (bottom - top + 192),
        ),
      );
      await session.hold(900);

      await session.click(symbol.locator(".pluto-input__switch-indicator").first(), {
        zoom: false,
      });
      await session.hold(400);
      // The command reaches the fixture, and its echo comes back, on wall time.
      await session.settleWall(800);
      await session.moveTo(aside);
      await session.hold(2400);
    } finally {
      await fixture.stop();
    }
  };
