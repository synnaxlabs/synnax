// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator, type Page } from "playwright";

import { capture, fixtures } from "@/index";

import { controls } from "./common/sample";

const AUTOMATION = "Tank pressurization";

const AUTOMATION_TEXT = `sequence pressurize {
    stage fill {
        tank_pt > 300 => hold
    }
    stage hold {}
}
`;

/** Milliseconds per character of a quick typist. */
const TYPE_MS = 45;

/** Frames the Arcs tree, the tabs, the breadcrumb, and the top of the editor. */
const WORK = { x: 560, y: 290 };

/** Empty editor space inside WORK, where the cursor rests clear of the code. */
const REST = { x: 900, y: 470 };

/** createFixtures creates the channels, the lib directory, and the automation. */
const createFixtures = async (): Promise<void> => {
  const client = fixtures.connect();
  try {
    const time = await client.channels.create({
      name: "tank_time",
      isIndex: true,
      dataType: "timestamp",
    });
    await client.channels.create([
      { name: "tank_pt", dataType: "float64", index: time.key },
      { name: "tank_pt_kpa", dataType: "float64", index: time.key },
    ]);
    const root = await client.arcs.directories.retrieveRoot();
    await client.arcs.directories.create({ parent: root.key, name: "lib" });
    const { key } = await client.arcs.create({
      name: AUTOMATION,
      type: "automation",
      source: { mode: "text", text: { raw: AUTOMATION_TEXT } },
    });
    const [rack] = await client.racks.retrieve({ integrations: "arc" });
    await client.arcs.updateTask(key, { rack: rack.key });
  } finally {
    await client.close();
  }
};

const path = (page: Page): Locator => page.locator(".console-arc-editor__path:visible");

const editor = (page: Page): Locator =>
  page.locator(".pluto-mosaic__leaf:visible .monaco-editor").first();

/**
 * firstLine returns a point on the editor's first line, past its text. A click on a
 * word would open its hover card.
 */
const firstLine = async (page: Page) => {
  const box = await editor(page).locator(".view-line").first().boundingBox();
  if (box == null) throw new Error("the first line of the Arc editor is hidden");
  return { x: box.x + 400, y: box.y + box.height / 2 };
};

/** lastLine returns a point on the editor's last line, past its text. */
const lastLine = async (page: Page) => {
  const boxes = await editor(page)
    .locator(".view-line")
    .evaluateAll((lines) =>
      lines.map((l) => {
        const r = l.getBoundingClientRect();
        return { x: r.x, y: r.y, height: r.height };
      }),
    );
  const last = boxes.reduce((a, b) => (b.y > a.y ? b : a));
  return { x: last.x + 400, y: last.y + last.height / 2 };
};

/**
 * Video for the Arc modules section of the v0.61 release notes: create a module named
 * tank, move it into the lib directory, and export a function from it. Then the Tank
 * pressurization automation imports lib.tank from the editor's completion and calls
 * the function in a flow.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await createFixtures();
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await capture.openToolbar(session, "arc");
  const lib = capture.treeItem(page, "arc_directory:", "lib");
  const automation = capture.treeItem(page, "arc:", AUTOMATION);
  await session.waitFor(automation);
  await session.moveTo(capture.PARK);

  session.startRecording();
  await session.hold(1600);

  await session.zoom(WORK, 1.6);
  await capture.clickToolbarCreate(session);
  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Arc name");
  await session.waitFor(name);
  await session.zoom(modal, 1.6);
  await session.hold(400);
  await session.type("tank");
  await session.hold(500);
  await session.click(modal.getByText("Module", { exact: true }).first(), {
    text: true,
  });
  await session.hold(1000);
  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(modal);
  await session.waitFor(editor(page));
  await session.hold(400);

  const tank = capture.treeItem(page, "arc:", "tank");
  await session.zoom(WORK, 1.6);
  await session.click(lib, { text: true });
  await session.hold(500);
  await session.drag(tank, lib, { zoom: false });
  await session.waitFor(path(page).getByText("lib", { exact: true }).first());
  await session.hold(500);

  await session.click(await firstLine(page));
  await session.moveTo(REST);
  await session.type("export func psi_to_kpa() {v f64} f64 {", TYPE_MS);
  await session.press("Enter");
  await session.type("return v * 6.895", TYPE_MS);
  await session.press("Escape");
  await session.hold(800);
  const compiled = controls(page).getByText("Compiled", { exact: true });
  await session.waitFor(compiled);
  await session.zoom(compiled, 1.6);
  await session.hold(1100);
  session.endZoom();

  await session.doubleClick(automation, { zoom: false, text: true });
  await session.waitFor(path(page).getByText(AUTOMATION, { exact: true }).first());
  await session.waitFor(controls(page).getByText("Node 1", { exact: true }));
  await session.hold(500);
  await session.zoom(WORK, 1.6);
  await session.click(await firstLine(page));
  await session.moveTo(REST);
  await session.press("Home");
  await session.press("Enter");
  await session.press("ArrowUp");
  await session.type("import lib.", 90);
  const suggestion = page
    .locator(".suggest-widget.visible .monaco-list-row")
    .filter({ hasText: "lib.tank" })
    .first();
  await session.waitFor(suggestion);
  await session.hold(1200);
  await session.press("Tab");
  await session.hold(300);
  await session.press("Enter");
  await session.hold(400);

  await session.click(await lastLine(page));
  await session.moveTo(REST);
  await session.press("Enter");
  await session.type("tank_pt -> tank.psi_to_kpa() -> tank_pt_kpa", TYPE_MS);
  await session.press("Escape");
  const errors = editor(page).locator(".squiggly-error");
  for (let i = 0; i < 600 && (await errors.count()) > 0; i++) await session.tick();
  if ((await errors.count()) > 0) throw new Error("the automation does not compile");
  await session.hold(1000);
  session.endZoom();
  await session.moveTo(capture.PARK);
  await session.hold(2600);
};
