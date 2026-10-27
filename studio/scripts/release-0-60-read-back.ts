// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic, type Synnax, TimeStamp } from "@synnaxlabs/client";
import { type Page } from "playwright";

import { capture, fixtures } from "@/index";

import { isolateTab, mosaicTab, NEW_PANEL, renamePanel } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

const PANEL = "Pressurization";
const NAME = "Tank press";
/** The setpoint's label, by which the script finds it on the canvas. */
const SETPOINT = "Press regulator";

/** The regulator's commanded outlet pressure, in psi. */
const COMMANDED = 300;
/** Live readings while the tank fills, in psi. */
const READINGS = { supply: 498.6, regulator: 287.4, tank: 276.8 };

/**
 * The part of the window the docs hero shows, in CSS pixels, less a margin. The
 * schematic draws inside it.
 */
const SHOWN = { left: 75, top: 100, right: 960, bottom: 575 };

/** Pipe runs between neighboring symbols, in canvas pixels. */
const RUN = { supply: 140, tank: 170 };
/** Space between the setpoint and the regulator under it, in canvas pixels. */
const GAP = 22;
/** Space between a vessel and the value under it, in canvas pixels. */
const VALUE_GAP = 22;

interface Channels {
  index: channel.Key;
  supply: channel.Key;
  regulator: channel.Key;
  tank: channel.Key;
  cmdIndex: channel.Key;
  cmd: channel.Key;
}

const createChannels = async (client: Synnax): Promise<Channels> => {
  const [index, cmdIndex] = await client.channels.create([
    { name: "press_time", isIndex: true, dataType: "timestamp" },
    { name: "press_reg_cmd_time", isIndex: true, dataType: "timestamp" },
  ]);
  const [supply, regulator, tank, cmd] = await client.channels.create([
    { name: "supply_pt", dataType: "float64", index: index.key },
    { name: "press_reg_pt", dataType: "float64", index: index.key },
    { name: "tank_pt", dataType: "float64", index: index.key },
    { name: "press_reg_cmd", dataType: "float64", index: cmdIndex.key },
  ]);
  return {
    index: index.key,
    supply: supply.key,
    regulator: regulator.key,
    tank: tank.key,
    cmdIndex: cmdIndex.key,
    cmd: cmd.key,
  };
};

type Config = NonNullable<schematic.New["configs"]>[string];

const label = (text: string) => ({ label: text, level: "h4" as const });

const value = (text: string, ch: channel.Key): Config => ({
  variant: "value",
  label: label(text),
  channel: ch,
  units: "psi",
  precision: 1,
  inlineSize: 100,
  level: "h3",
});

/**
 * feed draws a nitrogen bottle that feeds a tank through an electronic pressure
 * regulator, with a setpoint that commands the regulator and reads back its outlet
 * pressure. Every node starts at a rough position; layout places them once they draw.
 */
const feed = (ch: Channels): schematic.New => ({
  name: NAME,
  nodes: [
    { key: "supply", position: { x: 0, y: 100 } },
    { key: "supply_pt", position: { x: 0, y: 320 } },
    { key: "reg", position: { x: 230, y: 160 } },
    { key: "setpoint", position: { x: 200, y: 0 } },
    { key: "tank", position: { x: 520, y: 60 } },
    { key: "tank_pt", position: { x: 520, y: 340 } },
  ],
  edges: [
    {
      key: "supply_to_reg",
      source: { node: "supply", param: "8" },
      target: { node: "reg", param: "1" },
    },
    {
      key: "reg_to_tank",
      source: { node: "reg", param: "2" },
      target: { node: "tank", param: "7" },
    },
  ],
  configs: {
    supply: {
      variant: "tank",
      label: label("Nitrogen supply"),
      dimensions: { width: 90, height: 250 },
    },
    supply_pt: value("Supply pressure", ch.supply),
    reg: { variant: "electric_regulator", scale: 1.6 },
    setpoint: {
      variant: "setpoint",
      label: label(SETPOINT),
      size: "huge",
      inlineSize: 130,
      commandChannel: ch.cmd,
      stateChannel: ch.regulator,
      units: "psi",
      // Nobody holds control in the shot, so the control state widget says nothing.
      control: { hidden: true },
    },
    tank: {
      variant: "tank",
      label: label("Tank"),
      dimensions: { width: 150, height: 250 },
      channel: ch.tank,
      bounds: { lower: 0, upper: 500 },
    },
    tank_pt: value("Tank pressure", ch.tank),
    supply_to_reg: { variant: "pipe", segments: [] },
    reg_to_tank: { variant: "pipe", segments: [] },
  },
});

interface XY {
  x: number;
  y: number;
}

interface Box {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** A node as drawn, in canvas pixels from its position. */
interface Drawn {
  /** The box around the node and everything it draws, such as its label. */
  box: Box;
  /** The center of each handle, by handle key. */
  handles: Record<string, XY>;
}

const measure = async (page: Page, key: string): Promise<Drawn> =>
  await page.evaluate((k) => {
    const node = [
      ...document.querySelectorAll<HTMLElement>(`.react-flow__node[data-id="${k}"]`),
    ].find((n) => n.getBoundingClientRect().width > 0);
    if (node == null) throw new Error(`no node ${k} on screen`);
    const r = node.getBoundingClientRect();
    const scale = r.width / node.offsetWidth;
    let { left, top, right, bottom } = r;
    for (const el of node.querySelectorAll("*")) {
      if (el.classList.contains("react-flow__handle")) continue;
      const e = el.getBoundingClientRect();
      if (e.width === 0 || e.height === 0) continue;
      left = Math.min(left, e.left);
      top = Math.min(top, e.top);
      right = Math.max(right, e.right);
      bottom = Math.max(bottom, e.bottom);
    }
    const handles: Record<string, { x: number; y: number }> = {};
    for (const h of node.querySelectorAll<HTMLElement>(".react-flow__handle")) {
      const e = h.getBoundingClientRect();
      const id = h.dataset.handleid;
      if (id == null) continue;
      handles[id] = {
        x: (e.left + e.width / 2 - r.left) / scale,
        y: (e.top + e.height / 2 - r.top) / scale,
      };
    }
    return {
      box: {
        x: (left - r.left) / scale,
        y: (top - r.top) / scale,
        width: (right - left) / scale,
        height: (bottom - top) / scale,
      },
      handles,
    };
  }, key);

const handle = (drawn: Drawn, key: string): XY => {
  const h = drawn.handles[key];
  if (h == null) throw new Error(`no handle ${key} on the node`);
  return h;
};

/** onScreen returns the box around every drawn node, in CSS pixels. */
const onScreen = async (page: Page): Promise<Box> =>
  await page.evaluate(() => {
    let left = Infinity;
    let top = Infinity;
    let right = -Infinity;
    let bottom = -Infinity;
    for (const n of document.querySelectorAll(".react-flow__node *")) {
      if (n.classList.contains("react-flow__handle")) continue;
      const e = n.getBoundingClientRect();
      if (e.width === 0 || e.height === 0) continue;
      left = Math.min(left, e.left);
      top = Math.min(top, e.top);
      right = Math.max(right, e.right);
      bottom = Math.max(bottom, e.bottom);
    }
    return { x: left, y: top, width: right - left, height: bottom - top };
  });

/**
 * layout lines the bottle, the regulator, and the tank up on one pipe, puts the
 * setpoint over the regulator and each value under its vessel.
 */
const layout = async (page: Page, client: Synnax, key: string): Promise<void> => {
  const supply = await measure(page, "supply");
  const reg = await measure(page, "reg");
  const tank = await measure(page, "tank");
  const setpoint = await measure(page, "setpoint");
  const supplyPt = await measure(page, "supply_pt");
  const tankPt = await measure(page, "tank_pt");
  const outlet = handle(supply, "8");
  const supplyAt = { x: 0, y: 0 };
  const line = supplyAt.y + outlet.y;
  const regAt = {
    x: supplyAt.x + outlet.x + RUN.supply - handle(reg, "1").x,
    y: line - handle(reg, "1").y,
  };
  const regOutlet = regAt.x + handle(reg, "2").x;
  const tankAt = {
    x: regOutlet + RUN.tank - handle(tank, "7").x,
    y: line - handle(tank, "7").y,
  };
  const bottom = Math.max(
    supplyAt.y + supply.box.y + supply.box.height,
    tankAt.y + tank.box.y + tank.box.height,
  );
  // Both values sit on one line, under the lower vessel.
  const below = (at: XY, vessel: Drawn, v: Drawn): XY => ({
    x: at.x + vessel.box.x + vessel.box.width / 2 - v.box.x - v.box.width / 2,
    y: bottom + VALUE_GAP - v.box.y,
  });
  const moves: Array<[string, XY]> = [
    ["supply", supplyAt],
    ["reg", regAt],
    ["tank", tankAt],
    [
      "setpoint",
      {
        x:
          regAt.x +
          reg.box.x +
          reg.box.width / 2 -
          setpoint.box.x -
          setpoint.box.width / 2,
        y: regAt.y + reg.box.y - GAP - setpoint.box.y - setpoint.box.height,
      },
    ],
    ["supply_pt", below(supplyAt, supply, supplyPt)],
    ["tank_pt", below(tankAt, tank, tankPt)],
  ];
  const actions = moves.map(([k, position]) =>
    schematic.setNodePosition({ key: k, position }),
  );
  if (!(await client.schematics.dispatch(key, actions)))
    throw new Error("the schematic is not cached");
};

/**
 * pan moves the drawn schematic so it sits in the middle of the part of the window the
 * docs hero shows. Shift and drag pans a schematic outside edit mode.
 */
const pan = async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const drawn = await onScreen(page);
  const dx = (SHOWN.left + SHOWN.right) / 2 - (drawn.x + drawn.width / 2);
  const dy = (SHOWN.top + SHOWN.bottom) / 2 - (drawn.y + drawn.height / 2);
  const from = { x: 1100, y: 700 };
  await page.keyboard.down("Shift");
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(from.x + dx, from.y + dy, { steps: 12 });
  await page.mouse.up();
  await page.keyboard.up("Shift");
  await session.settle(400);
};

/**
 * Still for the setpoint read-back section of the v0.60 release notes: a nitrogen
 * bottle feeds a tank through an electronic pressure regulator. The regulator's
 * setpoint shows the 300 psi an operator commanded, and the tray under the field shows
 * the 287.4 psi the regulator reports while the tank fills.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const client = fixtures.connect();
  let reporting = true;
  let report: Promise<void> | undefined;
  try {
    await capture.login(session);
    await renamePanel(session, NEW_PANEL, PANEL);

    const ch = await createChannels(client);
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [ch.index, ch.supply, ch.regulator, ch.tank],
    });
    const jitter = (v: number, by: number) => v + (Math.random() - 0.5) * by;
    report = (async () => {
      while (reporting) {
        await writer.write({
          [ch.index]: [TimeStamp.now()],
          [ch.supply]: [jitter(READINGS.supply, 0.3)],
          [ch.regulator]: [jitter(READINGS.regulator, 0.04)],
          [ch.tank]: [jitter(READINGS.tank, 0.2)],
        });
        await new Promise((resolve) => setTimeout(resolve, 100));
      }
      await writer.close();
    })();

    const project = (
      await client.projects.retrieve({ searchTerm: capture.PROJECT })
    ).find((p) => p.name === capture.PROJECT);
    if (project == null) throw new Error(`no project named "${capture.PROJECT}"`);
    const { key } = await client.schematics.create(project.key, feed(ch));
    await capture.searchPalette(session, NAME);
    await session.waitFor(mosaicTab(session, NAME));
    const setpoint = page
      .locator(".react-flow__node")
      .filter({ has: page.getByText(SETPOINT, { exact: true }) })
      .first();
    await session.waitFor(setpoint);
    await isolateTab(session, NAME);
    await capture.hideBottomToolbar(session);
    const diagram = page.locator(".pluto-diagram:visible").first();
    await capture.control(diagram, "edit-off").click();
    await session.waitForHidden(diagram.locator(".react-flow__background"));
    await session.settle(600);
    await client.schematics.retrieve({ key });
    await layout(page, client, key);
    await session.settle(600);

    const command = await client.openWriter({
      start: TimeStamp.now(),
      channels: [ch.cmdIndex, ch.cmd],
    });
    await command.write({ [ch.cmdIndex]: [TimeStamp.now()], [ch.cmd]: [COMMANDED] });
    await command.close();
    const field = setpoint.locator("input").first();
    for (let tries = 0; Number(await field.inputValue()) !== COMMANDED; tries++) {
      if (tries > 300) throw new Error("the setpoint never showed the command");
      await session.settle(50);
    }

    await capture.fitView(session);
    await session.settle(600);
    await pan(session);
    const drawn = await onScreen(page);
    console.log("DRAWN", JSON.stringify(drawn));
    if (
      drawn.x < SHOWN.left - 20 ||
      drawn.y < SHOWN.top - 20 ||
      drawn.x + drawn.width > SHOWN.right + 20 ||
      drawn.y + drawn.height > SHOWN.bottom + 20
    )
      throw new Error(`the schematic overflows the hero: ${JSON.stringify(drawn)}`);

    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(1000);
    await session.settle(300);
    await session.still("window");
  } finally {
    reporting = false;
    await report;
    await client.close();
  }
};
