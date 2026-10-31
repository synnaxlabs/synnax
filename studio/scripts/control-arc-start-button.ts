// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

import { AUTOMATION, channels, createTankIndex } from "./common/press-sequence";

const PROGRAM = `import time

sequence press {
    stage pressurize {
        1 -> press_stage
        1 -> press_vlv_cmd
        tank_pt > 300 => next
    }
    2 -> press_stage
    0 -> press_vlv_cmd
    time.wait(duration=10s)
    stage vent {
        3 -> press_stage
        1 -> vent_vlv_cmd
        tank_pt < 5 => next
    }
    0 -> vent_vlv_cmd
    0 -> press_stage
}

start_cmd == 1 => press
`;

const STAGES = ["Idle", "Pressurizing", "Holding", "Venting"];

/** Creates the sequence's channels and writes Idle to `press_stage`. */
const createChannels = async (): Promise<void> => {
  const client = fixtures.connect();
  try {
    const tankTime = await createTankIndex(client);
    const stageTime = await client.channels.create(
      { name: "press_stage_time", isIndex: true, dataType: "timestamp" },
      { retrieveIfNameExists: true },
    );
    const [, , , , stage] = await client.channels.create(
      [
        ...channels(tankTime.key),
        { name: "press_stage", dataType: "uint8", index: stageTime.key },
      ],
      { retrieveIfNameExists: true },
    );
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [stageTime.key, stage.key],
    });
    await writer.write({ [stageTime.key]: TimeStamp.now(), [stage.key]: 0 });
    await writer.close();
  } finally {
    await client.close();
  }
};

/**
 * Docs `control/arc/start-from-schematic/start-button`: a schematic with the press
 * and vent valves, a Start button on `start_cmd`, and a state indicator on
 * `press_stage`, beside a running Arc sequence. A click on Start starts the sequence:
 * the indicator moves from Idle to Pressurizing, and the press valve opens.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await createChannels();
  const automation = await fixtures.runArc({ name: AUTOMATION, text: PROGRAM });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.createComponent(session, "Schematic");
    await session.waitFor(page.locator(".pluto-diagram").first());

    const properties = page.locator(".console-schematic__properties").first();
    const openTab = async (name: string): Promise<void> => {
      await session.click(
        properties.locator("[role='tab']").filter({ hasText: name }).first(),
        { text: true, zoom: false },
      );
    };
    const input = (label: string): Locator =>
      properties
        .locator(".lyra-input__item")
        .filter({ hasText: new RegExp(`^${label}`) })
        .locator("input")
        .first();
    const fill = async (label: string, value: string): Promise<void> => {
      const target = input(label);
      await session.waitFor(target);
      await target.fill(value);
      await target.press("Enter");
      await session.settle(300);
    };
    const valve = async (
      label: string,
      channel: string,
      at: capture.CanvasPosition,
    ): Promise<Locator> => {
      // "Generic" is the catalog name of the plain valve; "Valve" is its label.
      const symbol = await capture.place(session, "Generic", at);
      await openTab("Style");
      await fill("Label", label);
      await openTab("Control");
      await capture.bindChannel(session, "State", channel);
      await capture.bindChannel(session, "Command", channel);
      return symbol;
    };

    const press = await valve("Press valve", "press_vlv_cmd", { x: 620, y: 160 });
    const vent = await valve("Vent valve", "vent_vlv_cmd", { x: 750, y: 160 });

    const button = await capture.place(session, "Button", { x: 270, y: 160 });
    await openTab("Style");
    await fill("Label", "Start");
    await openTab("Control");
    await capture.bindChannel(session, "Command", "start_cmd");

    const indicator = await capture.place(session, "State indicator", {
      x: 400,
      y: 160,
    });
    await openTab("Telemetry");
    await capture.bindChannel(session, "State", "press_stage");
    await fill("Timeout", "60");
    await openTab("Options");
    await session.click(properties.getByText("Add an option", { exact: true }), {
      zoom: false,
    });
    for (let i = 1; i < STAGES.length; i++)
      await session.click(
        properties.getByRole("button", { name: "Add option" }).first(),
        { zoom: false },
      );
    const options = properties.locator(".pluto-state-mapping-list .lyra-list__item");
    for (const [i, name] of STAGES.entries()) {
      const nameInput = options.nth(i).locator("input").first();
      await session.waitFor(nameInput);
      await nameInput.fill(name);
      await session.settle(200);
    }

    await capture.deselect(session, { x: 40, y: 40 });
    await session.click(capture.control(page, "edit-off"), { zoom: false });
    await capture.hideBottomToolbar(session);
    await session.settle(600);
    await session.click(capture.control(page, "expand"), { zoom: false });
    await session.settle(600);
    const status = indicator.locator(".pluto-state-indicator").first();
    await session.waitFor(status.getByText("Idle", { exact: true }));
    await session.moveTo({ x: 800, y: 700 });

    const startButton = button.locator(".pluto-symbol-button").first();
    session.startRecording();
    await session.hold(1800);
    await session.zoom([button, indicator, press, vent], 1.6);
    await session.hold(1000);
    await session.click(startButton);
    session.endZoom();
    await session.waitFor(status.getByText("Pressurizing", { exact: true }));
    const body = await indicator.boundingBox();
    if (body == null) throw new Error("state indicator is hidden");
    await session.moveTo({
      x: body.x + body.width + 160,
      y: body.y + body.height + 120,
    });
    await session.hold(3000);
  } finally {
    await automation.stop();
  }
};
