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

/** sampleLink returns the empty projects screen's Open sample project link. */
export const sampleLink = (page: Page): Locator =>
  page.getByText("Open sample project", { exact: true }).first();

/** symbol returns the symbol of the sample's Tank schematic with the given label. */
export const symbol = (page: Page, label: string): Locator =>
  page
    .locator(".react-flow__node")
    .filter({ has: page.getByText(label, { exact: true }) })
    .first();

/** chip returns the control chip of a symbol. */
export const chip = (target: Locator): Locator =>
  target.locator(".pluto-control-chip").first();

/** absoluteChip matches a symbol's chip while the schematic holds absolute control. */
export const absoluteChip = (target: Locator): Locator =>
  target.locator(".pluto-control-chip[style*='secondary']").first();

/** lostChip matches a symbol's chip while a different writer controls its channel. */
export const lostChip = (target: Locator): Locator =>
  target.locator(".pluto-control-chip svg.lyra-icon--square").first();

/** toggle returns the part of a valve symbol that a click opens or closes. */
export const toggle = (target: Locator): Locator =>
  target.locator("button.pluto-symbol-primitive-toggle").first();

/** button returns the clickable part of the schematic's button with the given label. */
export const button = (page: Page, label: string): Locator =>
  symbol(page, label).locator(".pluto-symbol-button").first();

/**
 * openSample selects Open sample project on the empty projects screen of Synnax Desktop
 * and waits for the Tank schematic. Pass `zoom: false` through, since the click swaps
 * the view.
 */
export const openSample = async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const link = sampleLink(page);
  await session.waitFor(link, 1800);
  await session.click(link, { zoom: false });
  await session.waitFor(symbol(page, "Start"), 1800);
};

/** Options for openAutomation. */
export interface AutomationOptions {
  /** Name of the text-mode Arc automation to create. */
  name: string;
  /** Arc source code of the automation. */
  text: string;
  /** Starts the automation before it opens. */
  running?: boolean;
}

/**
 * openAutomation sets up the Arc tutorial off camera: it stops the sample's Press
 * sequence, creates an automation on the Core's Arc rack, and opens it in the editor.
 * The sample must be open.
 */
export const openAutomation = async (
  session: capture.CaptureSession,
  { name, text, running = false }: AutomationOptions,
): Promise<void> => {
  const { page } = session;
  const client = fixtures.connect();
  try {
    const press = await client.arcs.retrieve({ name: "Press sequence" });
    const pressTask = await client.arcs.task.retrieve(press.key);
    if (pressTask == null) throw new Error("the Press sequence has no task");
    await pressTask.stop();
    const { key } = await client.arcs.create({
      name,
      mode: "text",
      text: { raw: text },
    });
    if (running) await fixtures.startArc(client, key);
    else {
      const [rack] = await client.racks.retrieve({ integration: "arc" });
      await client.arcs.updateTask(key, { rack: rack.key });
    }
  } finally {
    await client.close();
  }
  await capture.openToolbar(session, "arc");
  const item = page
    .locator(".console-arc-toolbar")
    .getByText(name, { exact: true })
    .first();
  await session.waitFor(item);
  await session.doubleClick(item, { zoom: false, text: true });
  await session.waitFor(controls(page).getByText("Node 1", { exact: true }));
  await capture.closeToolbar(session);
};

/** tab returns the panel tab with the given name. */
export const tab = (page: Page, name: string): Locator =>
  page
    .locator(".console-mosaic .lyra-tabs__tab")
    .filter({ has: page.getByText(name, { exact: true }) })
    .first();

/** controls returns the controls bar of the open Arc editor. */
export const controls = (page: Page): Locator =>
  page.locator(".console-arc-editor__controls:visible").first();

/** editor returns the text editor of the open Arc automation. */
export const editor = (page: Page): Locator =>
  page.locator(".pluto-mosaic__leaf:visible .monaco-editor").first();

/** The sample tank as the capture core streams it. */
export interface Tank {
  /** Tank pressure in psi. */
  pressure: number;
  /** Press valve state: 1 is open. */
  press: number;
  /** Vent valve state: 1 is open. */
  vent: number;
  /** Closes the streamer and the client. */
  stop: () => Promise<void>;
}

/**
 * watchTank follows the sample's tank pressure and valve states, so a shot can wait on
 * the test instead of a fixed time. The sample must exist.
 */
export const watchTank = async (): Promise<Tank> => {
  const client = fixtures.connect();
  const streamer = await client.openStreamer([
    "sample_tank_pt",
    "sample_press_vlv_state",
    "sample_vent_vlv_state",
  ]);
  const latest = (frame: Awaited<ReturnType<typeof streamer.read>>, ch: string) =>
    Number(frame.get(ch).at(-1));
  const tank: Tank = {
    pressure: 0,
    press: 0,
    vent: 0,
    stop: async () => {
      streamer.close();
      await follow;
      await client.close();
    },
  };
  const follow = (async () => {
    for await (const frame of streamer) {
      const pressure = latest(frame, "sample_tank_pt");
      if (!Number.isNaN(pressure)) tank.pressure = pressure;
      const press = latest(frame, "sample_press_vlv_state");
      if (!Number.isNaN(press)) tank.press = press;
      const vent = latest(frame, "sample_vent_vlv_state");
      if (!Number.isNaN(vent)) tank.vent = vent;
    }
  })();
  return tank;
};
