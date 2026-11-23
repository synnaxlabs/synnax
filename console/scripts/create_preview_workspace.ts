// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Creates a workspace on a preview Core, so reviewers do not make data by hand. Run it
// after create_sample.ts, because it uses the live channels of the sample project. A
// second run replaces what the first one made.
//
// Usage: node scripts/create_preview_workspace.ts [PORT | URL], such as 9090
// (localhost) or https://core.example.com. The default is 9090.

import {
  type channel,
  lineplot,
  log,
  type panel,
  type resource,
  Synnax,
  table,
} from "@synnaxlabs/client";
import { dynamicWindow, TimeSpan } from "@synnaxlabs/x";

import { KEYS as SAMPLE } from "../src/feature/project/sample.ts";
import { parse } from "./coreURL.ts";

// Fixed keys let a second run replace the items of the first run.
const KEYS = {
  sampleLog: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e01",
  sampleTable: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e02",
  operations: {
    project: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e10",
    panel: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e11",
    linePlot: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e12",
    log: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e13",
    table: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e14",
  },
  review: {
    project: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e20",
    panel: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e21",
    linePlot: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e22",
    table: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e23",
  },
  valves: {
    project: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e30",
    panel: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e31",
    linePlot: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e32",
    log: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e33",
  },
} as const;

interface Channels {
  time: channel.Key;
  tankPT: channel.Key;
  pressVlv: channel.Key;
  ventVlv: channel.Key;
}

const retrieveChannels = async (client: Synnax): Promise<Channels> => {
  const names = [
    "sample_time",
    "sample_tank_pt",
    "sample_press_vlv_state",
    "sample_vent_vlv_state",
  ];
  const found = await client.channels.retrieve(names);
  const key = (name: string): channel.Key => {
    const ch = found.find((c) => c.name === name);
    if (ch == null) throw new Error(`Channel ${name} not found. Run create_sample.ts.`);
    return ch.key;
  };
  return {
    time: key(names[0]),
    tankPT: key(names[1]),
    pressVlv: key(names[2]),
    ventVlv: key(names[3]),
  };
};

const createLog = (key: string, name: string, ch: Channels): log.New => ({
  key,
  name,
  timestampPrecision: 2,
  channels: [
    { channel: ch.tankPT, alias: "Tank pressure", precision: 1 },
    { channel: ch.pressVlv, alias: "Press valve" },
    { channel: ch.ventVlv, alias: "Vent valve" },
  ],
});

const textCell = (key: string, value: string): table.Cell => ({
  key,
  config: table.textCellConfigZ.parse({ variant: "text", value, align: "start" }),
});

const valueCell = (key: string, channel: channel.Key, units = ""): table.Cell => ({
  key,
  config: table.valueCellConfigZ.parse({ variant: "value", channel, units }),
});

const createTable = (key: string, name: string, ch: Channels): table.New => ({
  key,
  name,
  columns: [
    { key: "label", size: 160 },
    { key: "value", size: 120 },
  ],
  rows: [
    {
      key: "tank_pt",
      cells: [
        textCell("tank_pt_label", "Tank pressure"),
        valueCell("tank_pt_value", ch.tankPT, "psi"),
      ],
    },
    {
      key: "press_vlv",
      cells: [
        textCell("press_vlv_label", "Press valve"),
        valueCell("press_vlv_value", ch.pressVlv),
      ],
    },
    {
      key: "vent_vlv",
      cells: [
        textCell("vent_vlv_label", "Vent valve"),
        valueCell("vent_vlv_value", ch.ventVlv),
      ],
    },
  ],
});

const createLinePlot = (
  key: string,
  name: string,
  y1: channel.Key[],
  window: TimeSpan,
  ch: Channels,
): lineplot.New => ({
  key,
  name,
  channels: { x1: ch.time, y1 },
  ranges: [{ variant: "window", window: dynamicWindow(window) }],
});

const leaf = (...resources: resource.ID[]): panel.New["root"] => ({
  variant: "leaf",
  tabs: resources.map((resource): panel.NewTab => ({ variant: "resource", resource })),
});

const createProject = async (
  client: Synnax,
  key: string,
  name: string,
): Promise<void> => {
  const found = await client.projects.retrieve({ key: { eq: key } });
  if (found.length > 0) await client.projects.rename(key, name);
  else await client.projects.create({ key, name });
};

const createSampleAdditions = async (client: Synnax, ch: Channels): Promise<void> => {
  const project = SAMPLE.project;
  await client.logs.create({ ...createLog(KEYS.sampleLog, "Tank log", ch), project });
  await client.tables.create({
    ...createTable(KEYS.sampleTable, "Tank readout", ch),
    project,
  });
};

const createOperations = async (client: Synnax, ch: Channels): Promise<void> => {
  const k = KEYS.operations;
  await createProject(client, k.project, "Tank operations");
  const project = k.project;
  const plot = createLinePlot(
    k.linePlot,
    "Pressure, last 2 minutes",
    [ch.tankPT],
    TimeSpan.minutes(2),
    ch,
  );
  await client.lineplots.create({ ...plot, project });
  await client.logs.create({ ...createLog(k.log, "Operations log", ch), project });
  await client.tables.create({ ...createTable(k.table, "Live values", ch), project });
  await client.panels.create({
    key: k.panel,
    name: "Tank operations",
    project,
    root: {
      variant: "split",
      direction: "x",
      size: 0.6,
      first: leaf(lineplot.resourceID(k.linePlot)),
      last: {
        variant: "split",
        direction: "y",
        size: 0.4,
        first: leaf(table.resourceID(k.table)),
        last: leaf(log.resourceID(k.log)),
      },
    },
  });
};

const createReview = async (client: Synnax, ch: Channels): Promise<void> => {
  const k = KEYS.review;
  await createProject(client, k.project, "Test review");
  const project = k.project;
  const plot = createLinePlot(
    k.linePlot,
    "Pressure, last 10 minutes",
    [ch.tankPT],
    TimeSpan.minutes(10),
    ch,
  );
  await client.lineplots.create({ ...plot, project });
  await client.tables.create({ ...createTable(k.table, "Summary", ch), project });
  await client.panels.create({
    key: k.panel,
    name: "Test review",
    project,
    root: leaf(lineplot.resourceID(k.linePlot), table.resourceID(k.table)),
  });
};

const createValves = async (client: Synnax, ch: Channels): Promise<void> => {
  const k = KEYS.valves;
  await createProject(client, k.project, "Valve health");
  const project = k.project;
  const plot = createLinePlot(
    k.linePlot,
    "Valve states",
    [ch.pressVlv, ch.ventVlv],
    TimeSpan.seconds(30),
    ch,
  );
  await client.lineplots.create({ ...plot, project });
  await client.logs.create({ ...createLog(k.log, "Valve log", ch), project });
  await client.panels.create({
    key: k.panel,
    name: "Valve health",
    project,
    root: {
      variant: "split",
      direction: "y",
      size: 0.5,
      first: leaf(lineplot.resourceID(k.linePlot)),
      last: leaf(log.resourceID(k.log)),
    },
  });
};

const deleteOld = async (client: Synnax): Promise<void> => {
  const { operations: o, review: r, valves: v } = KEYS;
  await Promise.all([
    client.panels.delete([o.panel, r.panel, v.panel]),
    client.lineplots.delete([o.linePlot, r.linePlot, v.linePlot]),
    client.logs.delete([KEYS.sampleLog, o.log, v.log]),
    client.tables.delete([KEYS.sampleTable, o.table, r.table]),
  ]);
};

const client = new Synnax({
  ...parse(process.argv[2] ?? process.env.SYNNAX_TEST_PORT ?? "9090"),
  username: "synnax",
  password: "seldon",
});
const ch = await retrieveChannels(client);
await deleteOld(client);
await createSampleAdditions(client, ch);
await createOperations(client, ch);
await createReview(client, ch);
await createValves(client, ch);
await client.close();
