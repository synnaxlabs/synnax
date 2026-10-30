// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { isolateTab, mosaicTab } from "./common/panels";

const NAME = "Pressurant Feed";
const PURPLE = "#b300ff";
const VALVE = "booster-iso";

type Config = NonNullable<schematic.New["configs"]>[string];

/** Symbols draw this much larger than their default size, to read on camera. */
const SCALE = 1.4;

const label = (text: string) => ({ label: text, level: "h5" as const });

/**
 * feed draws the TPC stand's pressurant feed: the supply bottle, the Gas Booster ISO
 * valve, the gas booster, the pressurant bottle, and the OX Press Iso valve that feeds
 * the OX tank on another page. The Gas Booster ISO valve is a generic one, which shares
 * its size and pipe height with a ball valve.
 */
const feed = (key: (name: string) => channel.Key): schematic.New => {
  const tank: Config = {
    variant: "tank",
    label: label(""),
    dimensions: { width: 200, height: 340 },
    strokeColor: PURPLE,
    levelHidden: true,
  };
  const value = (text: string, ch: string, units: string, inlineSize = 90): Config => ({
    variant: "value",
    label: label(text),
    channel: key(ch),
    units,
    strokeColor: PURPLE,
    level: "h4",
    inlineSize,
  });
  const valve = (variant: "valve" | "solenoid_valve", text: string, ch: string) =>
    ({
      variant,
      label: label(text),
      scale: SCALE,
      strokeColor: PURPLE,
      stateChannel: key(`${ch}_state`),
      commandChannel: key(`${ch}_cmd`),
      control: {},
    }) as Config;
  const nodes: Array<[string, { x: number; y: number }, Config]> = [
    ["supply", { x: 0, y: 120 }, tank],
    ["supply-pt", { x: 33, y: 290 }, value("Supply PT", "supply_pt", "psi", 100)],
    [VALVE, { x: 300, y: 265 }, valve("valve", "Gas Booster ISO", "gas_booster_iso")],
    [
      "booster",
      { x: 530, y: 252.7 },
      {
        variant: "piston_pump",
        label: label("Gas Booster"),
        scale: SCALE,
        strokeColor: PURPLE,
      },
    ],
    ["press", { x: 740, y: 120 }, tank],
    ["press-pt", { x: 778, y: 240 }, value("Press PT 1", "press_pt_1", "psi")],
    ["press-tc", { x: 779, y: 320 }, value("Press TC 1", "press_tc_1", "°C")],
    [
      "press-iso",
      { x: 1040, y: 234.2 },
      valve("solenoid_valve", "OX Press Iso", "press_iso"),
    ],
    [
      "ox-tank",
      { x: 1270, y: 274.5 },
      { variant: "off_page_reference", label: label("OX Tank"), fillColor: PURPLE },
    ],
  ];
  const pipes: Array<[string, string, string, string]> = [
    ["supply", "8", VALVE, "1"],
    [VALVE, "2", "booster", "1"],
    ["booster", "2", "press", "7"],
    ["press", "8", "press-iso", "1"],
    ["press-iso", "2", "ox-tank", "2"],
  ];
  const configs: Record<string, Config> = {};
  for (const [k, , config] of nodes) configs[k] = config;
  const edges = pipes.map(([source, sp, target, tp]) => {
    const k = `${source}-${target}`;
    configs[k] = { variant: "pipe", strokeColor: PURPLE, segments: [] };
    return {
      key: k,
      source: { node: source, param: sp },
      target: { node: target, param: tp },
    };
  });
  return {
    name: NAME,
    nodes: nodes.map(([k, position]) => ({ key: k, position })),
    edges,
    configs,
  };
};

/**
 * Docs `releases/0-60-0/change-symbol`: in the TPC stand's pressurant feed, select the
 * Gas Booster ISO valve, click Change, and pick the ball valve. The valve redraws as a
 * ball valve in place, with its label, channels, color, and pipes.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const tpc = (await client.projects.retrieve({ searchTerm: world.project })).find(
      (p) => p.name === world.project,
    );
    if (tpc == null) throw new Error(`no project named "${world.project}"`);
    const names = [
      "supply_pt",
      "press_pt_1",
      "press_tc_1",
      "gas_booster_iso_state",
      "gas_booster_iso_cmd",
      "press_iso_state",
      "press_iso_cmd",
    ];
    const keys = new Map(
      (await client.channels.retrieve(names)).map((ch) => [ch.name, ch.key]),
    );
    const key = (name: string): channel.Key => {
      const k = keys.get(name);
      if (k == null) throw new Error(`the stand has no channel "${name}"`);
      return k;
    };
    await client.schematics.create(tpc.key, feed(key));

    await capture.login(session, world.project);
    await session.click(page.getByText("Operations", { exact: true }).first());
    await session.waitFor(page.locator(".pluto-diagram:visible").first());
    await session.settle(1500);
    await capture.searchPalette(session, NAME);
    await session.waitFor(mosaicTab(session, NAME));
    await isolateTab(session, NAME);
    // The control legend lists the Arc that holds the valves, which is not the subject.
    await capture.selectToolbarTab(session, "Control");
    const legend = page.locator(".console-schematic__control input[type=checkbox]");
    await session.waitFor(legend.locator(".."));
    if (await legend.isChecked()) await legend.click({ force: true });
    await capture.selectToolbarTab(session, "Symbols");
    await capture.fitView(session);
    await session.settleWall(1500);
    await session.moveTo({ x: 26, y: 600 });
    await session.settle(800);

    const valve = page.locator(`.react-flow__node[data-id="${VALVE}"]`);
    const drawer = page
      .locator(".console-nav__drawer.lyra--visible.lyra--location-bottom")
      .first();

    // The valve and the left of the toolbar, where the change happens.
    const body = await valve.boundingBox();
    const bar = await drawer.boundingBox();
    if (body == null || bar == null) throw new Error("the valve or toolbar is hidden");
    const area = {
      left: bar.x,
      right: Math.max(body.x + body.width + 60, bar.x + 640),
      top: body.y - 60,
      bottom: bar.y + bar.height,
    };
    const fit = Math.min(
      1600 / (area.right - area.left),
      900 / (area.bottom - area.top),
    );

    session.startRecording();
    await session.hold(1800);

    await session.zoom(
      { x: (area.left + area.right) / 2, y: (area.top + area.bottom) / 2 },
      Math.min(1.6, fit * 0.92),
    );
    await session.hold(300);
    await session.click(valve.locator(".pluto-drag-handle").first());
    const change = drawer.getByRole("button", { name: "Change symbol" }).first();
    await session.waitFor(change);
    await session.hold(700);

    await session.click(change, { text: true });
    const group = drawer
      .locator(".lyra-tabs__tab")
      .filter({ hasText: "Valves" })
      .first();
    await session.waitFor(group);
    await session.hold(600);
    await session.click(group, { text: true });
    const ball = drawer
      .locator(".console-schematic-symbols__button")
      .filter({ has: page.getByText("Ball", { exact: true }) })
      .first();
    await session.waitFor(ball);
    await session.hold(700);
    await session.click(ball);
    session.endZoom();
    await session.hold(900);
    // Deselected, the valve drops its selection box and rotate control.
    const pane = await capture.canvas(page).boundingBox();
    if (pane == null) throw new Error("schematic canvas is not on screen");
    await session.click(
      { x: pane.x + pane.width / 2, y: pane.y + pane.height - 60 },
      {
        zoom: false,
      },
    );

    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(800);
    await session.hold(2800);
  } finally {
    await client.close();
    await world.stop();
  }
};
