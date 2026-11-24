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
  channel,
  DataType,
  group,
  type label,
  lineplot,
  log,
  type panel,
  ranger,
  type resource,
  status,
  Synnax,
  table,
  user,
} from "@synnaxlabs/client";
import { dynamicWindow, TimeSpan, TimeStamp } from "@synnaxlabs/x";

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
  groups: {
    channels: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e40",
    liveViews: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e41",
    reports: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e42",
  },
  labels: {
    hotFire: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e50",
    coldFlow: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e51",
    nominal: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e52",
    anomaly: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e53",
  },
  ranges: [
    "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e60",
    "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e61",
    "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e62",
  ],
  arcs: {
    bar: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e70",
    alarm: "0d4f4c0b-5c39-4a8e-9f3c-6b1c2f7a0e71",
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

// The Core refuses to delete a group that holds members, so a second run keeps it.
const createGroup = async (
  client: Synnax,
  key: group.Key,
  name: string,
  parent?: resource.ID,
): Promise<void> => {
  const found = await client.groups.retrieve({ key: { eq: key } });
  if (found.length > 0) return;
  await client.groups.create({ key, name, parent, scope: parent?.type ?? "channel" });
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
  const g = KEYS.groups.liveViews;
  await createGroup(client, g, "Live views", { type: "project", key: project });
  await client.lineplots.create({ ...plot, project, group: g });
  await client.logs.create({
    ...createLog(k.log, "Operations log", ch),
    project,
    group: g,
  });
  await client.tables.create({
    ...createTable(k.table, "Live values", ch),
    project,
  });
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
  const g = KEYS.groups.reports;
  await createGroup(client, g, "Reports", { type: "project", key: project });
  await client.lineplots.create({ ...plot, project, group: g });
  await client.tables.create({
    ...createTable(k.table, "Summary", ch),
    project,
    group: g,
  });
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

const createChannelGroup = async (client: Synnax, ch: Channels): Promise<void> => {
  const key = KEYS.groups.channels;
  await createGroup(client, key, "Sample tank");
  const ids = [ch.time, ch.tankPT, ch.pressVlv, ch.ventVlv].map((k) =>
    channel.resourceID(k),
  );
  await client.batch.send(client.groups.moveRequests(ids, group.resourceID(key)));
};

const createLabels = async (client: Synnax): Promise<void> => {
  const k = KEYS.labels;
  await client.labels.delete(Object.values(k));
  await client.labels.create([
    { key: k.hotFire, name: "Hot fire", color: { type: "fixed", value: "#E8590C" } },
    { key: k.coldFlow, name: "Cold flow", color: { type: "fixed", value: "#1C7ED6" } },
    { key: k.nominal, name: "Nominal", color: { type: "fixed", value: "#2F9E44" } },
    { key: k.anomaly, name: "Anomaly", color: { type: "fixed", value: "#E03131" } },
  ]);
};

interface Test {
  key: ranger.Key;
  name: string;
  hoursAgo: number;
  color: string;
  labels: label.Key[];
}

// A line plot draws only the ranges that have a color.
const PHASES = [
  { name: "Pressurize", color: "#F59F00" },
  { name: "Hold", color: "#7048E8" },
  { name: "Vent", color: "#0CA678" },
];

const createRanges = async (client: Synnax): Promise<void> => {
  const l = KEYS.labels;
  const [first, second, third] = KEYS.ranges;
  const tests: Test[] = [
    {
      key: first,
      name: "Hot fire 1",
      hoursAgo: 6,
      color: "#E8590C",
      labels: [l.hotFire, l.nominal],
    },
    {
      key: second,
      name: "Hot fire 2",
      hoursAgo: 3,
      color: "#F08C00",
      labels: [l.hotFire, l.anomaly],
    },
    {
      key: third,
      name: "Cold flow 1",
      hoursAgo: 1,
      color: "#1C7ED6",
      labels: [l.coldFlow, l.nominal],
    },
  ];
  // A range delete also deletes the ranges nested in it.
  await client.ranges.delete([...KEYS.ranges]);
  const now = TimeStamp.now();
  for (const test of tests) {
    const start = now.sub(TimeSpan.hours(test.hoursAgo));
    await client.ranges.create({
      key: test.key,
      name: test.name,
      color: { type: "fixed", value: test.color },
      timeRange: { start, end: start.add(TimeSpan.minutes(30)) },
    });
    await client.labels.label(ranger.resourceID(test.key), test.labels);
    await client.ranges.create(
      PHASES.map(({ name, color }, i): ranger.New => {
        const phaseStart = start.add(TimeSpan.minutes(10 * i));
        return {
          name,
          parent: test.key,
          color: { type: "fixed", value: color },
          timeRange: { start: phaseStart, end: phaseStart.add(TimeSpan.minutes(10)) },
        };
      }),
    );
  }
};

const STATUSES: status.New[] = [
  {
    key: "preview-workspace-ready",
    name: "Preview workspace",
    variant: "success",
    message: "Created preview workspace",
  },
  {
    key: "preview-tank-pressure",
    name: "Tank",
    variant: "warning",
    message: "Tank pressure near limit",
    description: "The tank pressure is above 450 psi. Vent the tank.",
  },
  {
    key: "preview-vent-valve",
    name: "Vent valve",
    variant: "error",
    message: "Vent valve did not close",
    description: "The vent valve state did not change in 2 seconds. Check the valve.",
  },
];

const BAR_PROGRAM = `import time

func convert() {
    preview_tank_pt_bar = sample_tank_pt * 0.0689476
}

time.interval(period=100ms) -> convert()
`;

const ALARM_PROGRAM = `import time

func check() {
    high u8 $= 0
    if sample_tank_pt > 400 {
        high = 1
    } else {
        high = 0
    }
    preview_tank_pt_high = high
}

time.interval(period=100ms) -> check()
`;

// The bar converter runs, and the alarm is configured but stopped, so the task toolbar
// shows both states.
const createArcs = async (client: Synnax): Promise<void> => {
  const k = KEYS.arcs;
  await client.channels.create(
    [
      { name: "preview_tank_pt_bar", dataType: DataType.FLOAT64, virtual: true },
      { name: "preview_tank_pt_high", dataType: DataType.UINT8, virtual: true },
    ],
    { retrieveIfNameExists: true },
  );
  // Deleting an Arc deletes its task, which stops the old program.
  await client.arcs.delete([k.bar, k.alarm]);
  await client.arcs.create([
    {
      key: k.bar,
      name: "Tank pressure in bar",
      type: "automation",
      source: { mode: "text", text: { raw: BAR_PROGRAM } },
    },
    {
      key: k.alarm,
      name: "Tank pressure alarm",
      type: "automation",
      source: { mode: "text", text: { raw: ALARM_PROGRAM } },
    },
  ]);
  const [rack] = await client.racks.retrieve({ integrations: "arc", embedded: true });
  if (rack == null) throw new Error("No rack can run Arc programs");
  const bar = await client.arcs.updateTask(k.bar, { rack: rack.key });
  await client.arcs.updateTask(k.alarm, { rack: rack.key });
  if (bar == null) throw new Error("The bar converter has no task");
  const state = await bar.executeCommandSync({ type: "start" });
  if (state.variant === "error") throw status.toError(state);
};

// Throwaway credentials for an in-memory preview. The preview comment lists them.
const USERS = [
  { username: "admin", name: "Ada Admin", role: "Owner" },
  { username: "engineer", name: "Eli Engineer", role: "Engineer" },
  { username: "operator", name: "Olga Operator", role: "Operator" },
  { username: "viewer", name: "Vic Viewer", role: "Viewer" },
];
const PASSWORD = "seldon";

const createUsers = async (client: Synnax): Promise<void> => {
  const roles = await client.access.roles.retrieve({});
  const roleKey = (name: string): string => {
    const role = roles.find((r) => r.name === name);
    if (role == null) throw new Error(`Role ${name} not found`);
    return role.key;
  };
  const usernames = USERS.map((u) => u.username);
  const found = await client.auth.retrievePasswords({ usernames });
  if (found.length > 0) await client.users.delete(found.map((p) => p.subject.key));
  const created = await client.users.create(
    USERS.map(({ name, role }) => ({ name, role: roleKey(role) })),
  );
  await Promise.all(
    created.map((u, i) =>
      client.auth.createPassword({
        subject: user.resourceID(u.key),
        username: USERS[i].username,
        password: PASSWORD,
      }),
    ),
  );
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
await createChannelGroup(client, ch);
await createLabels(client);
await createRanges(client);
await client.statuses.set(STATUSES);
await createArcs(client);
await createUsers(client);
await client.close();
