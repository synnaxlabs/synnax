// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { controls } from "./common/sample";

export const viewport = { width: 1280, height: 800 };

const AUTOMATION = "Chill-down hold";

// The line idles near -183 °C, so the loop never reaches its break.
const TEXT = `// Opens the oxidizer main valve once the line is cold enough to flow.
func chill_down() {line_temp f32} u8 {
    for {
        if line_temp < -190 {
            break
        }
    }
    return 1
}

ox_tc_1 -> chill_down() -> ox_mpv_cmd
`;

/**
 * Still for the Arc section of the v0.62 release notes: an automation that waits for
 * the oxidizer line to chill inside one cycle, with the Core's red Program ran too
 * long status in its controls bar, next to the open Arcs toolbar.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const { key } = await client.arcs.create({
      name: AUTOMATION,
      type: "automation",
      source: { mode: "text", text: { raw: TEXT } },
    });
    const [rack] = await client.racks.retrieve({ integrations: "arc" });
    await client.arcs.updateTask(key, { rack: rack.key });

    await capture.login(session, world.project);
    await session.click(page.getByText("Automations", { exact: true }).first());
    await session.waitFor(
      page.locator(".pluto-mosaic__leaf:visible .monaco-editor").first(),
    );
    await capture.openToolbar(session, "arc");
    const item = page
      .locator(".console-arc-toolbar")
      .getByText(AUTOMATION, { exact: true })
      .first();
    await session.waitFor(item);
    await session.doubleClick(item, { zoom: false, text: true });
    const bar = controls(page);
    await session.waitFor(bar.locator("button:has(svg.lyra-icon--play)"));
    await session.click(bar.locator("button:has(svg.lyra-icon--play)"));
    await session.waitFor(bar.getByText("Program ran too long", { exact: true }));
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
