// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type Synnax } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { isolateTab, mosaicTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

/** The part of the window the hero shows, in CSS pixels from its top left corner. */
const VISIBLE = { right: 981, bottom: 593 };

const NAME = "OX tank overpressure alarm";
const STATUS = "OX tank pressure";
/** The highest OX tank pressure that is normal, in psi. */
const LIMIT = 65;
const NORMAL = "OX tank pressure is normal";

/**
 * createAlarm creates the alarm as a graph Arc and returns its key. When either OX tank
 * transducer reads above the limit, the Arc sets the status to a warning, and back to
 * success when they do not.
 */
const createAlarm = async (client: Synnax): Promise<arc.Key> => {
  const ox1 = await client.channels.retrieve("ox_pt_1");
  const ox2 = await client.channels.retrieve("ox_pt_2");
  const { key: status } = await client.statuses.set({
    name: STATUS,
    variant: "success",
    message: NORMAL,
  });
  const inputs = {
    ox_pt_1: { type: "on", channel: ox1.key },
    limit_1: { type: "constant", value: LIMIT },
    ox_pt_2: { type: "on", channel: ox2.key },
    limit_2: { type: "constant", value: LIMIT },
    above_1: { type: "gt" },
    above_2: { type: "gt" },
    either: { type: "or" },
    select: { type: "select" },
    warning: {
      type: "status.set",
      key_or_name: status,
      variant: "warning",
      message: `OX tank pressure is above ${LIMIT} psi`,
    },
    normal: {
      type: "status.set",
      key_or_name: status,
      variant: "success",
      message: NORMAL,
    },
  };
  // Each operator sits midway between its two inputs, so no two lines cross.
  const positions: Record<keyof typeof inputs, [number, number]> = {
    ox_pt_1: [0, 0],
    limit_1: [0, 96],
    ox_pt_2: [0, 216],
    limit_2: [0, 312],
    above_1: [210, 38],
    above_2: [210, 254],
    either: [350, 146],
    select: [490, 146],
    warning: [630, 55],
    normal: [630, 255],
  };
  // Handle names match the ones the graph editor gives each block.
  const lines: Array<[string, string, string, string]> = [
    ["ox_pt_1", "output", "above_1", "a"],
    ["limit_1", "output", "above_1", "b"],
    ["ox_pt_2", "output", "above_2", "a"],
    ["limit_2", "output", "above_2", "b"],
    ["above_1", "output", "either", "a"],
    ["above_2", "output", "either", "b"],
    ["either", "output", "select", "output"],
    ["select", "true", "warning", "output"],
    ["select", "false", "normal", "output"],
  ];
  const { key } = await client.arcs.create({
    name: NAME,
    mode: "graph",
    graph: {
      nodes: Object.entries(positions).map(([node, [x, y]]) => ({
        key: node,
        position: { x, y },
      })),
      inputs,
      edges: lines.map(([from, output, to, input]) => ({
        key: `${from}-${to}-${input}`,
        source: { node: from, param: output },
        target: { node: to, param: input },
        kind: arc.ir.EdgeKind.continuous,
      })),
    },
  });
  return key;
};

/**
 * Still for the hero of docs page `reference/control/arc/graph-editor`: a graph Arc
 * that alarms on the TPC stand's OX tank pressure, running on the Core, alone in the
 * Automations panel with its blocks toolbar open. The graph and the task controls fit
 * inside the part of the window the hero shows.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await fixtures.startArc(client, await createAlarm(client));

    await capture.login(session, world.project);
    await session.click(page.getByText("Automations", { exact: true }).first());
    await session.settle(1000);
    const palette = await capture.openPalette(session);
    await session.type(NAME);
    await session.settle(800);
    // The Arc's task has the same name.
    await session.click(
      palette
        .locator(".lyra-list__item")
        .filter({ hasText: NAME, has: page.locator(".lyra-icon--arc") })
        .first(),
      { text: true },
    );
    await session.waitFor(mosaicTab(session, NAME));
    await isolateTab(session, NAME);
    await session.settle(1500);

    // With the bottom toolbar open, the panel ends just above the bottom of the part
    // the hero shows, so the task controls show in it.
    const drawer = page
      .locator(".console-nav__drawer.lyra--visible.lyra--location-bottom")
      .first();
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
    await session.waitFor(drawer);
    await session.settle(600);

    // The editor fit the graph to the whole panel when it opened, so a pan centers it
    // between the tab bar, the task controls, and the right edge of the hero.
    const pane = await capture.canvas(page).boundingBox();
    const controls = await page
      .locator(".console-arc-editor__controls")
      .first()
      .boundingBox();
    if (pane == null || controls == null) throw new Error("the graph is not on screen");
    const graph = {
      left: Infinity,
      top: Infinity,
      right: -Infinity,
      bottom: -Infinity,
    };
    for (const block of await page.locator(".react-flow__node").all()) {
      const box = await block.boundingBox();
      if (box == null) throw new Error("a block of the graph is not on screen");
      graph.left = Math.min(graph.left, box.x);
      graph.top = Math.min(graph.top, box.y);
      graph.right = Math.max(graph.right, box.x + box.width);
      graph.bottom = Math.max(graph.bottom, box.y + box.height);
    }
    const from = {
      x: (graph.right + pane.x + pane.width) / 2,
      y: (graph.top + graph.bottom) / 2,
    };
    const to = {
      x: from.x + (pane.x + VISIBLE.right - graph.left - graph.right) / 2,
      y: from.y + (pane.y + controls.y - graph.top - graph.bottom) / 2,
    };
    await page.keyboard.down("Shift");
    await session.drag(from, to, { zoom: false });
    await page.keyboard.up("Shift");

    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    // The graph's controls fade out once the cursor leaves it.
    await session.settle(600);
    await session.settleWall(1500);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
