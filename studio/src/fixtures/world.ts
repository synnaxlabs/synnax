// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

import {
  type channel,
  ni,
  project,
  type Synnax,
  task,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";
import { zip } from "@synnaxlabs/x";

import { connect, type ConnectionOptions } from "@/fixtures/client";

/** Project bundle of the test stand world, with channels referenced by name. */
const TPC_DIR = path.join(import.meta.dirname, "../../worlds/tpc");

/** The stand's coldflow sequence, an Arc program in the bundle. */
const SEQUENCE = "Ox Coldflow";

const SENSORS = [
  "ox_pt_1",
  "ox_pt_2",
  "fuel_pt_1",
  "fuel_pt_2",
  "press_pt_1",
  "press_pt_2",
  "supply_pt",
  "pneumatics_pt",
  "ox_tc_1",
  "ox_tc_2",
  "fuel_tc_1",
  "fuel_tc_2",
  "press_tc_1",
  "press_tc_2",
] as const;

type Sensor = (typeof SENSORS)[number];

/** Each valve's command and state channel prefix, with its label on the schematic. */
const VALVES = {
  ox_press: "OX TPC 1",
  ox_press_2: "OX TPC 2",
  ox_pre_valve: "OX Pre-Valve",
  purge: "Purge",
  ox_mpv: "OX MPV",
  ox_vent: "OX Vent",
  press_iso: "OX Press Iso",
  fuel_press_iso: "Fuel Press ISO",
  fuel_press: "Fuel TPC 1",
  fuel_press_2: "Fuel TPC 2",
  fuel_mpv: "Fuel MPV",
  fuel_pre_valve: "Fuel Pre-Valve",
  fuel_vent: "Fuel Vent",
  gas_booster_iso: "Gas Booster ISO",
} as const;

type Valve = keyof typeof VALVES;

const VALVE_NAMES = Object.keys(VALVES) as Valve[];

/** Vents that pass flow when de-energized, so state 1 closes them. */
const NORMALLY_OPEN = new Set<Valve>(["ox_vent", "fuel_vent"]);

/** Valves that cycle too often to log, since Arc regulates tank pressure with them. */
const UNLOGGED = new Set<Valve>([
  "ox_press",
  "ox_press_2",
  "fuel_press",
  "fuel_press_2",
]);

const START = "start_sim";
const LOG = "sequence_log";

/** Sample period of every sensor and valve state. */
const SAMPLE = TimeSpan.milliseconds(40);
/** Idle history written before the clock's current time, so plots open full. */
const HISTORY = TimeSpan.seconds(60);
/** Wall-clock interval at which the stand catches up to its clock. */
const POLL_MS = 10;
/** How long the Arc task has to report that it started. */
const START_TIMEOUT = TimeSpan.seconds(10);
/** Fuel tank pressure while the fuel side stands by, in psi. */
const FUEL_HOLD = 42;
/** Idle seconds recorded before and after each past run. */
const PAST_LEAD_S = 4;
const PAST_TAIL_S = 8;
/** A past run that has not safed by this many seconds never will. */
const PAST_MAX_S = 180;

/** Coldflow runs recorded before the capture, each saved as a range for review. */
const PAST_RUNS = [
  { name: "Coldflow 9", ago: TimeSpan.minutes(52), press: 172 },
  { name: "Coldflow 10", ago: TimeSpan.minutes(34), press: 186 },
  { name: "Coldflow 11", ago: TimeSpan.minutes(17), press: 180 },
] as const;

export interface WorldOptions extends ConnectionOptions {
  /**
   * The clock the stand samples on. A capture passes its virtual clock, so data arrives
   * at the same rate per video frame however slowly frames are captured.
   */
  now?: () => TimeStamp;
}

export interface World {
  /** Name of the imported project, for selection at login. */
  project: string;
  /** Every line the stand has written to its sequence log, oldest first. */
  logged: readonly string[];
  /** read returns a sensor's latest written value. Throws on an unknown sensor. */
  read: (sensor: string) => number;
  /** Stops the sequence and the simulation, closing every writer and the client. */
  stop: () => Promise<void>;
}

const noise = (amplitude: number): number =>
  amplitude * (Math.random() + Math.random() + Math.random() - 1.5);

/** approach moves value toward target with time constant tau seconds. */
const approach = (value: number, target: number, tau: number, dt: number): number =>
  value + (target - value) * (1 - Math.exp(-dt / tau));

/**
 * bind replaces names in a world file with the keys of what the world created: channel
 * names in `channel` fields, plot axis lists, and plot line keys, and `@`-prefixed
 * range names in plot range lists and line keys.
 */
const bind = (
  value: unknown,
  keys: Map<string, channel.Key>,
  ranges: Map<string, string>,
): unknown => {
  const resolve = (name: string): channel.Key => {
    const key = keys.get(name);
    if (key == null) throw new Error(`world file references unknown channel "${name}"`);
    return key;
  };
  const range = (name: string): string => {
    if (!name.startsWith("@")) return name;
    const key = ranges.get(name.slice(1));
    if (key == null) throw new Error(`world file references unknown range "${name}"`);
    return key;
  };
  if (Array.isArray(value)) return value.map((v) => bind(v, keys, ranges));
  if (value == null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([k, v]) => {
      if (k === "channel" && typeof v === "string") return [k, resolve(v)];
      if (/^y[1-4]$/.test(k) && Array.isArray(v))
        return [k, v.map((n) => (typeof n === "string" ? resolve(n) : n))];
      if (/^x[12]$/.test(k) && Array.isArray(v))
        return [k, v.map((n) => (typeof n === "string" ? range(n) : n))];
      if (k === "key" && typeof v === "string" && /^y[1-4]---/.test(v)) {
        const parts = v.split("---");
        parts[2] = range(parts[2]);
        parts[parts.length - 1] = String(resolve(parts[parts.length - 1]));
        return [k, parts.join("---")];
      }
      return [k, bind(v, keys, ranges)];
    }),
  );
};

/** Stand is the simulated state of the test stand's valves, tanks, and lines. */
interface Stand {
  state: Record<Valve, number>;
  p: { ox: number; fuel: number; press: number };
  t: { ox: number; fuel: number; press: number };
}

const newStand = (press: number): Stand => ({
  state: Object.fromEntries(VALVE_NAMES.map((v) => [v, 0])) as Record<Valve, number>,
  p: { ox: 0, fuel: FUEL_HOLD, press },
  t: { ox: -183, fuel: 19, press: 21 },
});

const flowing = (s: Stand, v: Valve): boolean =>
  NORMALLY_OPEN.has(v) ? s.state[v] === 0 : s.state[v] === 1;

/** step advances the stand by dt seconds and returns its sensor readings. */
const step = (s: Stand, dt: number): Record<Sensor, number> => {
  const { p, t } = s;
  const feeds = Number(flowing(s, "ox_press")) + Number(flowing(s, "ox_press_2"));
  const inflow = flowing(s, "press_iso")
    ? 0.25 * feeds * Math.max(0, p.press - p.ox)
    : 0;
  const vent = flowing(s, "ox_vent") ? 0.8 * p.ox : 0;
  const flow = flowing(s, "ox_mpv") ? 0.35 * p.ox : 0;
  const boost = flowing(s, "gas_booster_iso") ? 30 * Math.max(0, 1 - p.press / 320) : 0;
  p.ox = Math.max(0, p.ox + (inflow - vent - flow) * dt);
  p.press = Math.max(0, p.press + (boost - 2 * inflow) * dt);
  p.fuel = flowing(s, "fuel_vent") ? approach(p.fuel, 0, 1, dt) : p.fuel;
  t.ox = approach(t.ox, flow > 0 ? -190 : -183, 2, dt);
  t.press = approach(t.press, 21 - 0.1 * inflow, 1.5, dt);
  return {
    ox_pt_1: p.ox + noise(0.15),
    ox_pt_2: p.ox * 0.97 + noise(0.15),
    fuel_pt_1: p.fuel + noise(0.15),
    fuel_pt_2: p.fuel * 0.97 + noise(0.15),
    press_pt_1: p.press + noise(0.5),
    press_pt_2: p.press - 4 + noise(0.5),
    supply_pt: 310 + noise(0.8),
    pneumatics_pt: 108 + noise(0.3),
    ox_tc_1: t.ox + noise(0.1),
    ox_tc_2: t.ox + 0.6 + noise(0.1),
    fuel_tc_1: t.fuel + noise(0.1),
    fuel_tc_2: t.fuel + 0.3 + noise(0.1),
    press_tc_1: t.press + noise(0.1),
    press_tc_2: t.press - 0.4 + noise(0.1),
  };
};

/** Stage of the Ox Coldflow sequence, mirrored for runs recorded before the capture. */
type Stage = "idle" | "press" | "press_high" | "hold" | "tpc" | "safe";

/**
 * pastRun simulates one full Ox Coldflow run the way the stand's Arc program drives it,
 * with idle time on both sides, and returns its samples at the stand's sample period.
 */
const pastRun = (press: number): Array<Record<string, number>> => {
  const s = newStand(press);
  const dt = SAMPLE.seconds;
  const samples: Array<Record<string, number>> = [];
  const run: { stage: Stage; entered: number } = { stage: "idle", entered: 0 };
  for (let i = 0; run.stage !== "safe" || i * dt - run.entered < PAST_TAIL_S; i++) {
    const now = i * dt;
    if (now > PAST_MAX_S)
      throw new Error(`a past coldflow run never safed, stuck in stage ${run.stage}`);
    const enter = (next: Stage, cmds: Partial<Record<Valve, number>>): void => {
      Object.assign(s.state, cmds);
      run.stage = next;
      run.entered = now;
    };
    switch (run.stage) {
      case "idle":
        if (now >= PAST_LEAD_S)
          enter("press", { ox_vent: 1, press_iso: 1, ox_press: 1, gas_booster_iso: 1 });
        break;
      case "press":
        if (s.p.ox > 50) enter("press_high", { ox_press: 0, press_iso: 0 });
        break;
      case "press_high":
        if (s.p.press > 150) enter("hold", { gas_booster_iso: 0 });
        break;
      case "hold":
        if (now - run.entered >= 2) enter("tpc", { ox_mpv: 1, press_iso: 1 });
        break;
      case "tpc":
        if (s.p.ox < 15)
          enter("safe", { ox_press: 0, ox_mpv: 0, gas_booster_iso: 0, ox_vent: 0 });
        else if (s.p.ox > 25) s.state.ox_press = 0;
        else if (s.p.ox < 20) s.state.ox_press = 1;
        break;
      case "safe":
        break;
    }
    const sample: Record<string, number> = { ...step(s, dt) };
    for (const v of VALVE_NAMES) sample[`${v}_state`] = s.state[v];
    samples.push(sample);
  }
  return samples;
};

/** Key of the stand's DAQ card, an NI PXIe module. */
const DAQ_DEVICE = "01F3A8C2";

/**
 * hardware registers the stand's DAQ card with an analog read task that acquires every
 * sensor, and adds a Hardware panel holding the task's form to the project. The task is
 * never deployed, since the stand has no real card.
 */
const hardware = async (
  client: Synnax,
  proj: project.Key,
  daq: channel.Key,
  key: (name: string) => channel.Key,
): Promise<void> => {
  const rack = await client.racks.create({
    name: "Test stand DAQ",
    integrations: ["ni"],
  });
  await client.devices.create({
    key: DAQ_DEVICE,
    rack: rack.key,
    name: "NI PXIe-6363",
    make: "NI",
    model: "PXIe-6363",
    location: "PXI1Slot2",
    configured: true,
    properties: {
      is_simulated: false,
      resource_name: "PXI1Slot2",
      identifier: "daq",
      analogInput: {
        portCount: 32,
        index: daq,
        channels: Object.fromEntries(SENSORS.map((s, port) => [String(port), key(s)])),
      },
      analogOutput: { portCount: 4, stateIndex: 0, channels: {} },
      counterInput: { portCount: 4, index: 0, channels: {} },
      digitalInputOutput: { portCount: 3, lineCounts: [32, 8, 8] },
      digitalInput: { portCount: 3, lineCounts: [32, 8, 8], index: 0, channels: {} },
      digitalOutput: {
        portCount: 3,
        lineCounts: [32, 8, 8],
        stateIndex: 0,
        channels: {},
      },
    },
  });
  const channels = SENSORS.map((name, port) => {
    const base = {
      key: `ch${port}`,
      name,
      channel: key(name),
      port,
      device: DAQ_DEVICE,
    };
    if (!name.includes("_tc_"))
      return {
        ...base,
        type: "ai_voltage" as const,
        terminalConfig: "RSE" as const,
        minVal: 0,
        maxVal: 1000,
        // A 0.5 to 4.5 V transducer over 0 to 1000 psi.
        customScale: {
          type: "linear" as const,
          slope: 250,
          yIntercept: -125,
          preScaledUnits: "Volts" as const,
          scaledUnits: "psi",
        },
      };
    const cryogenic = name.startsWith("ox");
    return {
      ...base,
      type: "ai_thermocouple" as const,
      units: "DegC" as const,
      thermocoupleType: cryogenic ? ("T" as const) : ("K" as const),
      minVal: cryogenic ? -200 : -50,
      maxVal: cryogenic ? 50 : 150,
      cjc: { source: "const_val" as const, val: 25 },
    };
  });
  const acquire = await client.tasks.create({
    name: "Stand DAQ",
    type: "ni_analog_read",
    rack: rack.key,
    config: ni.analogReadConfigZ.parse({ sampleRate: 1000, streamRate: 25, channels }),
  });
  await client.panels.create({
    name: "Hardware",
    parent: project.ontologyID(proj),
    root: {
      variant: "leaf",
      tabs: [{ variant: "resource", resource: task.ontologyID(acquire.key) }],
    },
  });
};

/**
 * testStand builds an LOX coldflow stand on the capture's Core: its sensor, valve, and
 * log channels, three earlier coldflow runs saved as ranges, a DAQ card with its read
 * task, and the TPC project (panels for operations, each subsystem, automation,
 * hardware, and review). It deploys the Ox Coldflow
 * Arc program to the Core, where the Start Sequence button runs it, and simulates the
 * stand until stopped: valves follow their commands, and tank, pressurant, and feed
 * readings follow the valves.
 */
export const testStand = async ({
  now = () => TimeStamp.now(),
  ...opts
}: WorldOptions = {}): Promise<World> => {
  const client = connect(opts);
  const keys = new Map<string, channel.Key>();
  const key = (name: string): channel.Key => {
    const k = keys.get(name);
    if (k == null) throw new Error(`the stand has no channel "${name}"`);
    return k;
  };
  const create = async (specs: channel.New[]): Promise<void> => {
    for (const ch of await client.channels.create(specs)) keys.set(ch.name, ch.key);
  };
  const commanded = [...VALVE_NAMES, START];
  await create([
    { name: "daq_time", isIndex: true, dataType: "timestamp" },
    { name: `${LOG}_time`, isIndex: true, dataType: "timestamp" },
    // Arc stamps commands on wall time, so each command channel gets its own index.
    ...commanded.map((v) => ({
      name: `${v}_cmd_time`,
      isIndex: true,
      dataType: "timestamp",
    })),
  ]);
  const daq = key("daq_time");
  await create([
    ...SENSORS.map((name) => ({ name, dataType: "float32", index: daq })),
    ...VALVE_NAMES.map((v) => ({ name: `${v}_state`, dataType: "uint8", index: daq })),
    ...commanded.map((v) => ({
      name: `${v}_cmd`,
      dataType: "uint8",
      index: key(`${v}_cmd_time`),
    })),
    { name: LOG, dataType: "string", index: key(`${LOG}_time`) },
  ]);

  const stateChannels = [
    daq,
    ...SENSORS.map(key),
    ...VALVE_NAMES.map((v) => key(`${v}_state`)),
  ];
  const ranges = new Map<string, string>();
  for (const run of PAST_RUNS) {
    const samples = pastRun(run.press);
    const start = now().sub(run.ago);
    const stamps = samples.map((_, i) => start.add(SAMPLE.mult(i)));
    const series = new Map<channel.Key, number[]>([
      ...SENSORS.map((s): [channel.Key, number[]] => [
        key(s),
        samples.map((x) => x[s]),
      ]),
      ...VALVE_NAMES.map((v): [channel.Key, number[]] => [
        key(`${v}_state`),
        samples.map((x) => x[`${v}_state`]),
      ]),
    ]);
    const past = await client.openWriter({ start, channels: stateChannels });
    await past.write({ [daq]: stamps, ...Object.fromEntries(series) });
    await past.close();
    const range = await client.ranges.create({
      name: run.name,
      timeRange: { start, end: stamps[stamps.length - 1].add(SAMPLE) },
    });
    ranges.set(run.name, range.key);
  }

  const names = (await readdir(TPC_DIR)).filter((n) => n.endsWith(".json"));
  const files = await Promise.all(
    names.map(async (name): Promise<zip.Source> => {
      const raw = await readFile(path.join(TPC_DIR, name), "utf8");
      const bytes = new TextEncoder().encode(
        JSON.stringify(bind(JSON.parse(raw), keys, ranges)),
      );
      return { path: name, read: async () => bytes };
    }),
  );
  const bundle = new Uint8Array(await new Response(zip.create(files)).arrayBuffer());
  const imported = await client.projects.import(bundle, { fileName: "TPC.zip" });
  await hardware(client, imported.key, daq, key);

  const arc = await client.arcs.retrieve({ name: SEQUENCE });
  const [rack] = await client.racks.retrieve({ integration: "arc" });
  if (rack == null) throw new Error("the Core has no rack that runs Arc");
  const task = await client.arcs.updateTask(arc.key, { rack: rack.key });
  if (task == null)
    throw new Error(`binding ${SEQUENCE} to rack ${rack.name} made no task`);
  const started = await task.executeCommandSync({
    type: "start",
    timeout: START_TIMEOUT,
  });
  if (started.variant === "error")
    throw new Error(`${SEQUENCE} failed to start: ${started.message}`);

  let time = now().sub(HISTORY);
  const writer = await client.openWriter({ start: time, channels: stateChannels });
  const logWriter = await client.openWriter({
    start: time,
    channels: [key(`${LOG}_time`), key(LOG)],
  });

  const stand = newStand(180);
  /** Valve changes seen since the last sample, applied on the stand's clock. */
  const pending = new Map<Valve, number>();
  const byCommand = new Map<channel.Key, Valve>(
    VALVE_NAMES.map((v): [channel.Key, Valve] => [key(`${v}_cmd`), v]),
  );
  const streamer = await client.openStreamer([...byCommand.keys()]);
  const commands = (async () => {
    for await (const frame of streamer)
      for (const [k, v] of byCommand) {
        const series = frame.get(k);
        if (series.length > 0) pending.set(v, Number(series.at(-1)));
      }
  })();

  const logged: string[] = [];
  const latest = new Map<string, number>(SENSORS.map((s) => [s, 0]));
  let running = true;
  const sim = (async () => {
    while (running) {
      await new Promise((resolve) => setTimeout(resolve, POLL_MS));
      const target = now();
      const stamps: TimeStamp[] = [];
      const values = new Map<channel.Key, number[]>();
      const logStamps: TimeStamp[] = [];
      const logs: string[] = [];
      while (time.add(SAMPLE).beforeEq(target)) {
        time = time.add(SAMPLE);
        for (const [v, value] of pending) {
          if (stand.state[v] === value) continue;
          stand.state[v] = value;
          if (UNLOGGED.has(v)) continue;
          logStamps.push(time);
          logs.push(`${VALVES[v]} ${flowing(stand, v) ? "open" : "closed"}`);
        }
        pending.clear();
        const sample = step(stand, SAMPLE.seconds);
        stamps.push(time);
        const push = (k: channel.Key, v: number): void => {
          const series = values.get(k);
          if (series == null) values.set(k, [v]);
          else series.push(v);
        };
        for (const s of SENSORS) {
          push(key(s), sample[s]);
          latest.set(s, sample[s]);
        }
        for (const v of VALVE_NAMES) push(key(`${v}_state`), stand.state[v]);
      }
      if (stamps.length > 0)
        await writer.write({ [daq]: stamps, ...Object.fromEntries(values) });
      if (logs.length > 0) {
        await logWriter.write({ [key(`${LOG}_time`)]: logStamps, [key(LOG)]: logs });
        logged.push(...logs);
      }
    }
    await writer.close();
    await logWriter.close();
  })();

  return {
    project: imported.name,
    logged,
    read: (sensor) => {
      const value = latest.get(sensor);
      if (value == null) throw new Error(`the stand has no sensor "${sensor}"`);
      return value;
    },
    stop: async () => {
      await task.stop();
      running = false;
      streamer.close();
      await Promise.all([commands, sim]);
      await client.close();
    },
  };
};
