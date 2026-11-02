// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type table } from "@synnaxlabs/client";
import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

import { isolateTab, mosaicTab } from "./common/panels";

const NAME = "Stand Readings";

// Fills at partial opacity tint the cell on both themes, under text that stays legible.
const GREEN = "#38b26159";
const YELLOW = "#f4ca258c";
const RED = "#f5242e73";
const BLUE = "#356fc573";

const band = (threshold: number, color: string) => ({ threshold, color });

/** The units and redline bands of each kind of reading. */
const KINDS = {
  cryogenic: {
    units: "°C",
    bands: [band(-273, BLUE), band(-175, YELLOW), band(-160, RED)],
  },
  ambient: { units: "°C", bands: [band(-40, GREEN), band(40, YELLOW), band(60, RED)] },
  level: { units: "%", bands: [band(-1, RED), band(10, YELLOW), band(25, GREEN)] },
  pressurant: {
    units: "psi",
    bands: [
      band(-15, RED),
      band(30, YELLOW),
      band(60, GREEN),
      band(280, YELLOW),
      band(320, RED),
    ],
  },
};

type Reading = readonly [label: string, channel: string, kind: keyof typeof KINDS];

/** The stand's readings by subsystem, with the pressurant last. */
const READINGS: Reading[] = [
  ["OX TC 1", "ox_tc_1", "cryogenic"],
  ["OX Level", "ox_level", "level"],
  ["Fuel TC 1", "fuel_tc_1", "ambient"],
  ["Fuel Level", "fuel_level", "level"],
  ["Press PT 1", "press_pt_1", "pressurant"],
  ["Press TC 1", "press_tc_1", "ambient"],
];

/** The sensor, units, and value columns, with the units before the value. */
const COLUMNS = [
  { key: "sensor", size: 250 },
  { key: "units", size: 110 },
  { key: "value", size: 200 },
];
const HEADER_HEIGHT = 54;
const ROW_HEIGHT = 74;
/** A gray that reads as a faint tint on both themes. */
const HEADER_FILL = "#80808033";
/** A gray legible on both themes, a step quieter than the labels. */
const UNITS_COLOR = "#8a8f98";

type Cells = NonNullable<table.New["cells"]>;
type CellConfig = Cells[string];

/** readings lays out a header and each reading's label, units, and live value. */
const readings = (keys: Map<string, channel.Key>): table.New => {
  const cells: Cells = {};
  const add = (config: CellConfig): string => {
    const key = `cell-${Object.keys(cells).length}`;
    cells[key] = config;
    return key;
  };
  const header = {
    key: "header",
    size: HEADER_HEIGHT,
    cells: ["Sensor", "Units", "Value"].map((value) =>
      add({
        variant: "text",
        value,
        level: "h5",
        weight: 600,
        align: "start",
        fillColor: HEADER_FILL,
      }),
    ),
  };
  const rows = READINGS.map(([label, name, kind]) => {
    const key = keys.get(name);
    if (key == null) throw new Error(`the stand has no channel "${name}"`);
    const { units, bands } = KINDS[kind];
    return {
      key: name,
      size: ROW_HEIGHT,
      cells: [
        add({ variant: "text", value: label, level: "h4", align: "start" }),
        add({ variant: "text", value: units, level: "h4", textColor: UNITS_COLOR }),
        add({
          variant: "value",
          channel: key,
          precision: 1,
          level: "h3",
          redline: { bands: bands.map((b, i) => ({ key: `b${i}`, ...b })) },
        }),
      ],
    };
  });
  return { name: NAME, columns: COLUMNS, rows: [header, ...rows], cells };
};

/**
 * Docs `releases/0-60-0/table-reorder`: a table of the TPC stand's live readings. Drag
 * the Value column ahead of the Units column, then select the two pressurant rows and
 * drag them together to the top.
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
    const channels = await client.channels.retrieve(READINGS.map(([, name]) => name));
    await client.tables.create(
      tpc.key,
      readings(new Map(channels.map((ch) => [ch.name, ch.key]))),
    );

    await capture.login(session, world.project);
    await session.click(page.getByText("Operations", { exact: true }).first());
    await session.waitFor(page.locator(".pluto-diagram:visible").first());
    await session.settle(1500);
    await capture.searchPalette(session, NAME);
    await session.waitFor(mosaicTab(session, NAME));
    await isolateTab(session, NAME);
    const surface = page.locator(".console-mosaic .console-table:visible").first();
    await capture.contextMenu(session, surface, "Center table");
    session.endZoom();
    await capture.hideBottomToolbar(session);
    await session.settleWall(2000);
    await session.moveTo(capture.PARK);
    await session.settle(800);

    const label = (dir: "x" | "y", key: string): Locator =>
      page.locator(`[id="resizer-${dir}-${key}"] .pluto-table__label`).first();
    const center = async (target: Locator): Promise<{ x: number; y: number }> => {
      const box = await target.boundingBox();
      if (box == null) throw new Error("an indicator is not on screen");
      return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
    };

    session.startRecording();
    await session.hold(1800);

    await session.zoom(page.locator(".pluto-table").first());
    await session.hold(400);

    // Past the Units column, so the Value column lands second.
    const value = await center(label("x", "value"));
    await session.drag(
      value,
      { x: value.x - (COLUMNS[1].size + 10), y: value.y },
      { zoom: false },
    );
    await session.settleWall(600);
    await session.hold(800);

    await session.click(label("y", "press_pt_1"));
    await page.keyboard.down("Shift");
    await session.click(label("y", "press_tc_1"));
    await page.keyboard.up("Shift");
    await session.hold(600);

    // Up four readings, so the pair lands under the header.
    const press = await center(label("y", "press_pt_1"));
    await session.drag(
      press,
      { x: press.x, y: press.y - 4 * ROW_HEIGHT },
      { zoom: false },
    );
    session.endZoom();
    await session.settleWall(600);
    await session.hold(600);

    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await client.close();
    await world.stop();
  }
};
