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
 * Docs `console/schematics/control-tab`: select a valve, open the Control tab of its
 * properties, and pick its command channel and then its state channel.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.echoValve();
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Schematic");
    await session.waitFor(page.locator(".pluto-diagram").first());
    // "Generic" is the catalog name of the plain valve; "Valve" is its label.
    const valve = await capture.place(session, "Generic", { x: 500, y: 200 });
    await capture.deselect(session, { x: 100, y: 290 });
    await capture.fitView(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1800);

    await session.click(valve, { zoom: false });
    const properties = page.locator(".console-schematic__properties").first();
    await session.waitFor(properties);
    await session.hold(700);
    await capture.zoomCenter(session, properties, 1.4);
    await session.click(
      properties.locator("[role='tab']").filter({ hasText: "Control" }).first(),
      { text: true, zoom: false },
    );
    await session.hold(700);

    const bind = async (section: string, channel: string): Promise<void> => {
      const item = properties
        .locator(".lyra-form-section")
        .filter({
          has: page
            .locator(".lyra-form-section__header")
            .getByText(section, { exact: true }),
        })
        .locator(".lyra-input__item")
        .filter({ hasText: "Channel" })
        .first();
      await session.click(item.getByText("Select channel", { exact: true }).first(), {
        text: true,
        zoom: false,
      });
      const search = page.getByPlaceholder("Search channels...").first();
      await session.waitFor(search);
      await session.hold(500);
      session.setSpeed(1.5);
      await session.type(channel);
      session.setSpeed(1);
      const option = page
        .locator(".lyra-list__item:not(.lyra-tree__item)")
        .filter({ has: page.getByText(channel, { exact: true }) })
        .first();
      await session.waitFor(option);
      await session.hold(400);
      await session.click(option.getByText(channel, { exact: true }).first(), {
        text: true,
        zoom: false,
      });
      await session.hold(900);
    };
    await bind("Command", fixture.command);
    await bind("State", fixture.state);
    session.endZoom();

    await session.moveTo({ x: 300, y: 350 });
    await session.hold(3000);
  } finally {
    await fixture.stop();
  }
};
