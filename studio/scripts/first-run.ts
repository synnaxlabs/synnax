// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { kindButton, selectItem } from "./common/channel-dialog";
import { controls, editor } from "./common/sample";

export const desktop = true;

const PROGRAM = `import time

func step() {
    value f64 $= 0
    value += 1
    if value > 100 {
        value = 0
    }
    ramp = value
}

time.interval(period=100ms) -> step()
`;

/** Presentation speed of typed text: the tutorial plays at speed. */
const TYPE_SPEED = 2.5;

const type = async (session: capture.CaptureSession, text: string): Promise<void> => {
  session.setSpeed(TYPE_SPEED);
  await session.type(text);
  session.setSpeed(1);
};

/** openProject creates an empty project from the projects screen of Synnax Desktop. */
const openProject = async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const create = page.getByText("New project").first();
  await session.waitFor(create, 1800);
  await create.click();
  const name = page.locator(capture.MODAL).getByPlaceholder("Name").first();
  await session.waitFor(name);
  await name.fill("My project");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await session.waitFor(page.locator(".console-palette button"), 1800);
  await capture.clearPanel(session);
};

/**
 * Docs `get-started/first-run`: the whole tutorial at speed. Create the index channel
 * ramp_time and the channel ramp, start the Ramp Arc automation on the Core, plot ramp
 * on a rolling 30 s plot, and create the First run range from a selection on the plot.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openProject(session);
  // V toggles the toolbar, so it starts closed.
  await capture.hideBottomToolbar(session);
  await session.moveTo({ x: 800, y: 450 });
  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Name").first();

  session.startRecording();
  await session.hold(1200);

  await capture.commandPalette(session, "Create channel", { typeSpeed: TYPE_SPEED });
  await session.waitFor(name);
  await session.zoom(modal);
  await session.click(name);
  await type(session, "ramp_time");
  await session.click(kindButton(modal, "Index"));
  await session.hold(400);
  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(name);
  await session.hold(400);

  await capture.commandPalette(session, "Create channel", { typeSpeed: TYPE_SPEED });
  await session.waitFor(name);
  await session.zoom(modal);
  await session.click(name);
  await type(session, "ramp");
  await session.click(
    modal.locator(".lyra-dialog__trigger").filter({ hasText: "float32" }).first(),
  );
  await session.waitFor(page.getByPlaceholder("Search data types...").first());
  await session.hold(300);
  await type(session, "float64");
  await selectItem(session, "float64");
  await session.click(modal.getByText("Select channel", { exact: true }).first(), {
    text: true,
  });
  await selectItem(session, "ramp_time");
  await session.hold(400);
  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(name);
  await session.hold(400);

  await capture.commandPalette(session, "Create Arc automation", {
    typeSpeed: TYPE_SPEED,
  });
  await session.waitFor(name);
  await session.zoom(modal, 1.6);
  await session.click(name);
  await type(session, "Ramp");
  await session.click(
    modal
      .locator(".console-arc-create-modal__mode-select-button")
      .filter({ hasText: "Text" })
      .first(),
  );
  await session.hold(400);
  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(name);
  await session.waitFor(editor(page));
  await session.click(editor(page), { zoom: false });
  const box = await editor(page).boundingBox();
  if (box == null) throw new Error("the Arc editor is hidden");
  // The editor fills the panel, so frame its top left, where the program lands.
  await session.zoom({ x: box.x + 380, y: box.y + 200 }, 2);
  await session.paste(PROGRAM);
  await session.hold(1500);
  const rack = controls(page).locator(".console-rack-select").first();
  const play = controls(page).locator("button:has(svg.lyra-icon--play)");
  await session.zoom([rack, play], 1.8);
  await session.click(rack);
  await selectItem(session, "Node 1");
  await session.hold(400);
  await session.click(play);
  await session.waitFor(controls(page).getByText("Started task"));
  await session.hold(1000);
  session.endZoom();

  await capture.commandPalette(session, "Create line plot", { typeSpeed: TYPE_SPEED });
  const plot = page.locator(".pluto-mosaic__leaf:visible .pluto-line-plot").first();
  await session.waitFor(plot);
  await session.hold(600);
  await session.press("v");
  await session.waitFor(page.locator("label").filter({ hasText: "Y1" }).first());
  await session.hold(400);
  const drawer = page.locator(".console-nav__drawer.lyra--location-bottom").first();
  await capture.zoomCenter(session, drawer, 1.4);
  await capture.addChannels(session, "Y1", ["ramp"], { search: "ramp" });
  // A new plot starts on Rolling 30s, so the tutorial only checks it.
  await session.moveTo(drawer.getByText("Rolling 30s", { exact: true }).first(), {
    text: true,
  });
  await session.hold(1200);
  session.endZoom();
  await session.moveTo(capture.PARK);
  // The ramp draws on wall time, which the capture runs behind, so a few seconds of
  // video show more than one climb.
  await session.hold(4000);

  const plotBox = await plot.boundingBox();
  if (plotBox == null) throw new Error("the line plot is hidden");
  const at = (fx: number, fy: number): { x: number; y: number } => ({
    x: plotBox.x + plotBox.width * fx,
    y: plotBox.y + plotBox.height * fy,
  });
  await session.zoom(plot, 1.2);
  await page.keyboard.down("Alt");
  await session.drag(at(0.45, 0.35), at(0.75, 0.65));
  await page.keyboard.up("Alt");
  await session.hold(500);
  await session.rightClick(at(0.6, 0.5));
  const menu = page.locator(".lyra-menu-context").first();
  await session.waitFor(menu);
  await session.hold(400);
  await session.click(
    menu.getByText("Create range from selection", { exact: true }).first(),
    { text: true },
  );
  session.endZoom();
  await session.waitFor(name);
  await session.zoom(modal);
  await session.click(name);
  await type(session, "First run");
  await session.hold(400);
  await capture.clickButton(session, "Save to Core");
  session.endZoom();
  await session.waitForHidden(name);
  await session.moveTo(capture.PARK);
  await session.hold(2800);
};
