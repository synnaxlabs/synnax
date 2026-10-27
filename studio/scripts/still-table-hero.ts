// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type table, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { isolateTab, mosaicTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

const NAME = "Sensors";

// Fills at partial opacity tint the cell on both themes, under text that stays legible.
const GREEN = "#38b26159";
const YELLOW = "#f4ca258c";
const RED = "#f5242e73";
const BLUE = "#356fc573";

const band = (threshold: number, color: string) => ({ threshold, color });

/** The units and redline bands of each kind of reading. */
const KINDS = {
  tank: { units: "psi", bands: [band(-15, GREEN), band(55, YELLOW), band(70, RED)] },
  cryogenic: {
    units: "°C",
    bands: [band(-273, BLUE), band(-175, YELLOW), band(-160, RED)],
  },
  ambient: { units: "°C", bands: [band(-40, GREEN), band(40, YELLOW), band(60, RED)] },
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
  supply: {
    units: "psi",
    bands: [band(-15, RED), band(150, GREEN), band(300, YELLOW), band(340, RED)],
  },
  pneumatics: {
    units: "psi",
    bands: [band(-15, RED), band(90, GREEN), band(125, YELLOW), band(150, RED)],
  },
};

type Reading = readonly [label: string, channel: string, kind: keyof typeof KINDS];

interface Section {
  name: string;
  readings: Reading[];
}

/**
 * The stand's sensors by subsystem, two sections side by side. The fuel tank vents
 * during an OX coldflow, so its pressures stay out.
 */
const SECTIONS: ReadonlyArray<readonly [Section, Section]> = [
  [
    {
      name: "Oxidizer",
      readings: [
        ["OX PT 1", "ox_pt_1", "tank"],
        ["OX PT 2", "ox_pt_2", "tank"],
        ["OX TC 1", "ox_tc_1", "cryogenic"],
        ["OX TC 2", "ox_tc_2", "cryogenic"],
      ],
    },
    {
      name: "Pressurant",
      readings: [
        ["Press PT 1", "press_pt_1", "pressurant"],
        ["Press PT 2", "press_pt_2", "pressurant"],
        ["Press TC 1", "press_tc_1", "ambient"],
        ["Press TC 2", "press_tc_2", "ambient"],
      ],
    },
  ],
  [
    {
      name: "Fuel",
      readings: [
        ["Fuel TC 1", "fuel_tc_1", "ambient"],
        ["Fuel TC 2", "fuel_tc_2", "ambient"],
      ],
    },
    {
      name: "Facility",
      readings: [
        ["Supply PT", "supply_pt", "supply"],
        ["Pneumatics PT", "pneumatics_pt", "pneumatics"],
      ],
    },
  ],
];

/** Widths of the sensor, value, and units columns of each half of the table. */
const COLUMNS = [200, 180, 90];
const SECTION_HEIGHT = 48;
const ROW_HEIGHT = 82;
/** A gray that reads as a faint tint on both themes. */
const SECTION_FILL = "#80808033";
/** A gray legible on both themes, a step quieter than the labels. */
const UNITS_COLOR = "#8a8f98";

type Cells = NonNullable<table.New["cells"]>;
type CellConfig = Cells[string];

/** sensorTable lays out every reading with its label, live value, and units. */
const sensorTable = (keys: Map<string, channel.Key>): table.New => {
  const cells: Cells = {};
  const add = (config: CellConfig): string => {
    const key = `cell-${Object.keys(cells).length}`;
    cells[key] = config;
    return key;
  };
  const reading = ([label, name, kind]: Reading): string[] => {
    const key = keys.get(name);
    if (key == null) throw new Error(`the stand has no channel "${name}"`);
    const { units, bands } = KINDS[kind];
    return [
      add({ variant: "text", value: label, level: "h4", align: "start" }),
      add({
        variant: "value",
        channel: key,
        precision: 1,
        level: "h3",
        redline: {
          bands: bands.map((b, i) => ({ key: `b${i}`, ...b })),
        },
      }),
      add({ variant: "text", value: units, level: "h4", textColor: UNITS_COLOR }),
    ];
  };
  const rows = SECTIONS.flatMap((pair) => [
    {
      size: SECTION_HEIGHT,
      cells: pair.flatMap(({ name }) =>
        [name, "", ""].map((value) =>
          add({
            variant: "text",
            value,
            level: "h5",
            weight: 600,
            align: "start",
            fillColor: SECTION_FILL,
          }),
        ),
      ),
    },
    ...pair[0].readings.map((left, i) => ({
      size: ROW_HEIGHT,
      cells: [...reading(left), ...reading(pair[1].readings[i])],
    })),
  ]);
  return {
    name: NAME,
    columns: [...COLUMNS, ...COLUMNS].map((size) => ({ size })),
    rows,
    cells,
  };
};

/**
 * Still for the hero of docs page `reference/visualize/tables`: a table of the TPC
 * stand's sensors by subsystem, alone in the Operations panel mid-coldflow, with
 * editing and indicators off. The hero shows only the top left of the window, which
 * the table fills.
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
    const names = SECTIONS.flat().flatMap((s) => s.readings.map(([, name]) => name));
    const channels = await client.channels.retrieve(names);
    await client.tables.create(
      tpc.key,
      sensorTable(new Map(channels.map((ch) => [ch.name, ch.key]))),
    );

    await capture.login(session, world.project);
    await session.click(page.getByText("Operations", { exact: true }).first());
    await session.waitFor(page.locator(".pluto-diagram:visible").first());
    await session.settle(1500);
    await capture.searchPalette(session, NAME);
    await session.waitFor(mosaicTab(session, NAME));
    await isolateTab(session, NAME);
    const surface = page.locator(".console-mosaic .console-table:visible").first();
    // Indicators show while editing, so editing goes off before they hide.
    for (const option of ["Disable editing", "Hide indicators"]) {
      await capture.contextMenu(session, surface, option);
      session.endZoom();
      await session.settle(300);
    }

    // Starts the Ox Coldflow sequence, as the schematic's Start Sequence button does.
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ["start_sim_cmd_time", "start_sim_cmd"],
    });
    await writer.write({ start_sim_cmd_time: TimeStamp.now(), start_sim_cmd: 1 });
    await writer.close();
    await session.waitFor(() => world.logged.includes("OX MPV open"), 3600);
    // Partway through the coldflow, the OX side is chilled and the pressurant draining.
    await session.settle(3000);
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    // Live values draw on wall time, which virtual ticks outrun.
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};
