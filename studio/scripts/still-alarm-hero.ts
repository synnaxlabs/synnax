// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  status,
  type table,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { isolateTab, mosaicTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

const NAME = "Limits";

// Fills at partial opacity tint the cell on both themes, under text that stays legible.
const YELLOW = "#f4ca258c";
const RED = "#f5242e73";
const CLEAR = "#00000000";
/** A gray that reads as a faint tint on both themes. */
const HEADER_FILL = "#80808033";
/** A gray legible on both themes, a step quieter than the labels. */
const QUIET = "#8a8f98";

/**
 * A reading and its limits. A high reading warns at or above `warning`, and a low one
 * below it.
 */
type Limit = readonly [
  label: string,
  channel: string,
  units: string,
  kind: "high" | "low",
  warning: number,
  error: number,
];

/** The readings the stand's alarms watch, from the facility supply to the OX tank. */
const LIMITS: Limit[] = [
  ["Purge PT", "pneumatics_pt", "psi", "low", 120, 110],
  ["Supply PT", "supply_pt", "psi", "high", 300, 340],
  ["Press PT 1", "press_pt_1", "psi", "low", 60, 30],
  ["OX PT 1", "ox_pt_1", "psi", "high", 55, 70],
  ["OX PT 2", "ox_pt_2", "psi", "high", 55, 70],
  ["OX TC 1", "ox_tc_1", "°C", "high", -175, -160],
  ["OX TC 2", "ox_tc_2", "°C", "high", -175, -160],
];

/** bands leaves a value clear inside its limits. */
const bands = ([, , , kind, warning, error]: Limit) =>
  (kind === "high"
    ? [
        { threshold: warning, color: YELLOW },
        { threshold: error, color: RED },
      ]
    : [
        { threshold: -15, color: RED },
        { threshold: error, color: YELLOW },
        { threshold: warning, color: CLEAR },
      ]
  ).map((b, i) => ({ key: `b${i}`, ...b }));

/** The stand's subsystems, which label its alarms. */
const LABELS = {
  Facility: "#ffa800",
  Oxidizer: "#3774d0",
  Pressurant: "#a34ff0",
  Fuel: "#e8457b",
  Engine: "#2fb6a6",
};

interface Alarm {
  name: string;
  variant: status.Variant;
  message: string;
  description?: string;
  label: keyof typeof LABELS;
  /** How long before the still the alarm last changed. */
  ago: TimeSpan;
}

/**
 * The alarms on the stand, most severe first. Each message agrees with the limits of
 * the reading the alarm watches.
 */
const ALARMS: Alarm[] = [
  {
    name: "Purge supply",
    variant: "error",
    message: "Below 110 psi",
    description: "Hold the test and check the purge regulator.",
    label: "Facility",
    ago: TimeSpan.seconds(47),
  },
  {
    name: "Supply pressure",
    variant: "warning",
    message: "Above 300 psi",
    description: "Lower the facility regulator before the next run.",
    label: "Facility",
    ago: TimeSpan.seconds(252),
  },
  {
    name: "OX tank pressure",
    variant: "success",
    message: "Within limits",
    label: "Oxidizer",
    ago: TimeSpan.seconds(570),
  },
  {
    name: "OX feed line",
    variant: "success",
    message: "Chilled down",
    label: "Oxidizer",
    ago: TimeSpan.seconds(96),
  },
  {
    name: "Pressurant bottle",
    variant: "success",
    message: "Within limits",
    label: "Pressurant",
    ago: TimeSpan.minutes(14),
  },
  {
    name: "Fuel tank",
    variant: "info",
    message: "Vented",
    label: "Fuel",
    ago: TimeSpan.minutes(22),
  },
  {
    name: "Igniter",
    variant: "info",
    message: "Disarmed",
    label: "Engine",
    ago: TimeSpan.minutes(38),
  },
];

const alarmKey = (name: string): string =>
  `alarm:${name.toLowerCase().replaceAll(" ", "-")}`;

/** alarms returns the stand's alarms as statuses, each changed its `ago` before now. */
const alarms = (now: TimeStamp): status.New[] =>
  ALARMS.map(({ name, variant, message, description, ago }) => ({
    key: alarmKey(name),
    name,
    variant,
    message,
    description,
    time: now.sub(ago),
  }));

/** Widths of the sensor, value, units, warning, and error columns. */
const COLUMNS = [170, 150, 70, 120, 120];
const HEADER_HEIGHT = 48;
const ROW_HEIGHT = 78;

type Cells = NonNullable<table.New["cells"]>;
type CellConfig = Cells[string];

/** limitTable lays out every reading with its live value, units, and limits. */
const limitTable = (keys: Map<string, channel.Key>): table.New => {
  const cells: Cells = {};
  const add = (config: CellConfig): string => {
    const key = `cell-${Object.keys(cells).length}`;
    cells[key] = config;
    return key;
  };
  const quiet = (value: string): string =>
    add({ variant: "text", value, level: "h5", textColor: QUIET });
  const header = {
    size: HEADER_HEIGHT,
    cells: ["Sensor", "Value", "", "Warning", "Error"].map((value, i) =>
      add({
        variant: "text",
        value,
        level: "h5",
        weight: 600,
        align: i === 0 ? "start" : "center",
        fillColor: HEADER_FILL,
      }),
    ),
  };
  const rows = LIMITS.map((limit) => {
    const [label, name, units, kind, warning, error] = limit;
    const key = keys.get(name);
    if (key == null) throw new Error(`the stand has no channel "${name}"`);
    const sign = kind === "high" ? ">" : "<";
    return {
      size: ROW_HEIGHT,
      cells: [
        add({ variant: "text", value: label, level: "h4", align: "start" }),
        add({
          variant: "value",
          channel: key,
          precision: 1,
          level: "h3",
          redline: { bands: bands(limit) },
        }),
        quiet(units),
        quiet(`${sign} ${warning}`),
        quiet(`${sign} ${error}`),
      ],
    };
  });
  return {
    name: NAME,
    columns: COLUMNS.map((size) => ({ size })),
    rows: [header, ...rows],
    cells,
  };
};

/**
 * Still for the hero of docs page `reference/alarms`: the statuses toolbar open on the
 * TPC stand's alarms mid-coldflow, beside a table of the readings they watch. Values
 * past a limit take the color of the alarm they set. The hero shows only the top left
 * of the window, which the toolbar and the table fill.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    // The time since each change counts from the page's clock.
    const pageNow = async (): Promise<TimeStamp> =>
      TimeStamp.milliseconds(await page.evaluate(() => Date.now()));
    const tpc = (await client.projects.retrieve({ searchTerm: world.project })).find(
      (p) => p.name === world.project,
    );
    if (tpc == null) throw new Error(`no project named "${world.project}"`);
    const channels = await client.channels.retrieve(LIMITS.map(([, name]) => name));
    await client.tables.create(
      tpc.key,
      limitTable(new Map(channels.map((ch) => [ch.name, ch.key]))),
    );
    await client.statuses.set(alarms(await pageNow()));
    const labels = new Map(
      (
        await client.labels.create(
          Object.entries(LABELS).map(([name, color]) => ({ name, color })),
        )
      ).map((l) => [l.name, l.key]),
    );
    for (const { name, label } of ALARMS) {
      const key = labels.get(label);
      if (key == null) throw new Error(`no label named "${label}"`);
      await client.labels.label(status.ontologyID(alarmKey(name)), [key]);
    }

    await capture.login(session, world.project);
    await favorite(
      session,
      ALARMS.map((a) => a.name),
    );
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
    // Opened before the coldflow, so the table settles at its final size.
    await capture.openToolbar(session, "status");

    // Starts the Ox Coldflow sequence, as the schematic's Start Sequence button does.
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ["start_sim_cmd_time", "start_sim_cmd"],
    });
    await writer.write({ start_sim_cmd_time: TimeStamp.now(), start_sim_cmd: 1 });
    await writer.close();
    await session.waitFor(() => world.logged.includes("OX MPV open"), 3600);
    await session.settle(3000);

    // Set again, so the time since each change reads as the alarm history says.
    await client.statuses.set(alarms(await pageNow()));
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    // Live values draw on wall time, which virtual ticks outrun.
    await session.settleWall(2500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
    await world.stop();
  }
};

/**
 * favorite opens the status explorer from the palette, favorites each named status in
 * it, and closes the explorer's tab.
 */
const favorite = async (
  session: capture.CaptureSession,
  names: readonly string[],
): Promise<void> => {
  const { page } = session;
  await capture.commandPalette(session, "Open status explorer");
  for (const name of names) {
    const item = page.locator(".lyra-list__item").filter({ hasText: name }).first();
    await session.waitFor(item);
    await capture.contextMenu(
      session,
      item.getByText(name, { exact: true }).first(),
      "Favorite",
    );
    session.endZoom();
    await session.settle(300);
  }
  const tab = page.locator(".console-mosaic .lyra-tabs__tab.lyra--selected").first();
  await session.click(tab.locator(".lyra-tabs__close").first(), { zoom: false });
  await session.settle(500);
};
