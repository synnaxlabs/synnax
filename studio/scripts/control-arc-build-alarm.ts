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

const STATUS = "Tank pressure";

/** Creates `tank_pt` and the status the alarm sets. */
const createResources = async (): Promise<void> => {
  const client = fixtures.connect();
  try {
    const time = await client.channels.create(
      { name: "tank_time", isIndex: true, dataType: "timestamp" },
      { retrieveIfNameExists: true },
    );
    await client.channels.create(
      { name: "tank_pt", dataType: "float64", index: time.key },
      { retrieveIfNameExists: true },
    );
    await client.statuses.set({
      name: STATUS,
      variant: "success",
      message: "Tank pressure is normal",
    });
  } finally {
    await client.close();
  }
};

/**
 * Docs `control/arc/graph-editor/build-alarm`: in an empty graph-mode Arc automation,
 * drag a telemetry source, a constant, a greater-than operator, a select, and two
 * change-status blocks onto the graph, set their properties, connect them, choose the
 * Driver, and start the automation.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await createResources();
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await capture.clickPanelCreate(session);
  await capture.createComponent(session, "Arc automation");
  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Name").first();
  await session.waitFor(name);
  await name.fill("Tank pressure alarm");
  await session.click(
    modal
      .locator(".console-arc-create-modal__mode-select-button")
      .filter({ hasText: "Graph" })
      .first(),
    { zoom: false },
  );
  await capture.clickButton(session, "Create");
  await session.waitForHidden(modal);
  const pane = capture.canvas(page);
  await session.waitFor(pane);

  const drawer = page.locator(".console-nav__drawer.lyra--location-bottom").first();
  const stagesTab = drawer.getByText("Stages", { exact: true }).first();
  if (!(await stagesTab.isVisible().catch(() => false))) {
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
    await session.waitFor(stagesTab);
  }
  await session.settle(600);

  const nodes = page.locator(".react-flow__node");
  // Positions are fractions of the canvas, so the layout fills it.
  const at = async (x: number, y: number) => {
    const box = await pane.boundingBox();
    if (box == null) throw new Error("graph canvas is not on screen");
    return { x: box.x + x * box.width, y: box.y + y * box.height };
  };
  const group = async (id: string): Promise<void> => {
    await session.click(drawer.locator(`button[id='${id}']`).first(), {
      text: true,
      zoom: false,
    });
    await session.hold(300);
  };
  const add = async (block: string, x: number, y: number): Promise<Locator> => {
    const before = await nodes.count();
    await session.drag(
      drawer.locator(`[aria-label='${block}']`).first(),
      await at(x, y),
      { zoom: false },
    );
    const node = nodes.nth(before);
    await session.waitFor(node);
    await session.hold(300);
    return node;
  };

  // The graph fits its view once, when the first block lands, which would move the
  // first on-camera block. A throwaway block takes that fit.
  const throwaway = await add("Constant", 0.4, 0.3);
  await session.settle(1000);
  await session.click(throwaway, { zoom: false });
  await session.press("Backspace");
  await session.waitForHidden(throwaway);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1800);

  // Blocks keep clear of the canvas edges, where a connection drag auto-pans.
  await group("telem");
  const source = await add("Telemetry source", 0.1, 0.18);
  await group("basic");
  const constant = await add("Constant", 0.1, 0.5);
  await group("operator");
  const greater = await add("Greater than", 0.3, 0.36);
  await group("flow_control");
  const select = await add("Select", 0.46, 0.36);
  await group("basic");
  const warning = await add("Change status", 0.64, 0.18);
  const success = await add("Change status", 0.64, 0.52);
  await session.hold(500);

  const properties = drawer.locator(".console-arc-properties__individual").first();
  const configure = async (node: Locator): Promise<void> => {
    await session.click(node, { zoom: false });
    const tab = drawer.getByText("Properties", { exact: true }).first();
    if (!(await properties.isVisible().catch(() => false)))
      await session.click(tab, { text: true, zoom: false });
    await session.waitFor(properties);
    await session.hold(300);
  };
  const typeInto = async (target: Locator, text: string): Promise<void> => {
    await session.click(target, { zoom: false });
    await session.press("ControlOrMeta+a");
    session.setSpeed(2);
    await session.type(text);
    session.setSpeed(1);
    await session.press("Enter");
    await session.hold(400);
  };
  const choose = async (trigger: Locator, item: string, search?: string) => {
    await session.click(trigger, { zoom: false });
    if (search != null) {
      const input = page.getByPlaceholder(search).first();
      await session.waitFor(input);
      session.setSpeed(2);
      await session.type(item);
      session.setSpeed(1);
    }
    const option = page
      .locator(".lyra-list__item")
      .filter({ has: page.getByText(item, { exact: true }) })
      .first();
    await session.waitFor(option);
    await session.hold(300);
    await session.click(option, { text: true, zoom: false });
    await session.hold(500);
  };
  const field = (label: string): Locator =>
    properties
      .locator(".lyra-input__item")
      .filter({ hasText: new RegExp(`^${label}`) });

  await capture.zoomCenter(session, drawer, 1.4);
  await configure(source);
  await choose(
    properties.getByText("Select channel", { exact: true }).first(),
    "tank_pt",
    "Search channels...",
  );

  await configure(constant);
  await typeInto(properties.locator("input").first(), "450");

  const setStatus = async (
    node: Locator,
    variant: string | null,
    message: string,
  ): Promise<void> => {
    await configure(node);
    await choose(field("Status").locator("button").first(), STATUS);
    if (variant != null)
      await choose(field("Variant").locator("button").first(), variant);
    await typeInto(field("Message").locator("input").first(), message);
  };
  await setStatus(warning, "Warning", "Tank pressure is above 450 psi");
  await setStatus(success, null, "Tank pressure is normal");
  session.endZoom();
  await capture.deselect(session, await at(0.9, 0.4));
  await session.hold(600);

  // Handles sit behind the block body, so only the part of their marker outside the
  // block takes a press.
  const exposed = async (handle: Locator): Promise<{ x: number; y: number }> => {
    const found = await handle.evaluate((el) => {
      const r = el.getBoundingClientRect();
      for (let dx = -10; dx <= 16; dx += 1)
        for (let dy = -6; dy <= 8; dy += 1) {
          const x = r.x + r.width / 2 + dx;
          const y = r.y + r.height / 2 + dy;
          const hit = document.elementFromPoint(x, y);
          if (hit != null && (hit === el || el.contains(hit))) return { x, y };
        }
      return null;
    });
    if (found == null) throw new Error("handle is covered");
    return found;
  };
  const connect = async (from: Locator, to: Locator): Promise<void> => {
    await session.drag(await exposed(from), await exposed(to), { zoom: false });
    await session.hold(400);
  };
  await connect(capture.handle(source, "right"), capture.handle(greater, "left", 0));
  await connect(capture.handle(constant, "right"), capture.handle(greater, "left", 1));
  await connect(capture.handle(greater, "right"), capture.handle(select, "left"));
  await connect(capture.handle(select, "right", 0), capture.handle(warning, "left"));
  await connect(capture.handle(select, "right", 1), capture.handle(success, "left"));
  await session.hold(800);

  const controls = page.locator(".console-arc-editor__controls").first();
  await capture.zoomCenter(
    session,
    controls.locator(".console-rack-select").first(),
    1.5,
  );
  await choose(controls.locator(".console-rack-select").first(), "Node 1");
  await session.click(controls.locator("button:has(svg.lyra-icon--play)").first());
  session.endZoom();
  await session.waitFor(controls.getByText("Started task").first());
  await session.moveTo(await at(0.9, 0.4));
  await session.hold(3000);
};
