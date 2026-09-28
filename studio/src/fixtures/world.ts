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

import { type channel, TimeSpan, TimeStamp } from "@synnaxlabs/client";
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
 * bind replaces channel names in a world file with the keys of the created channels:
 * `channel` fields, plot axis lists, and plot line keys.
 */
const bind = (value: unknown, keys: Map<string, channel.Key>): unknown => {
  const resolve = (name: string): channel.Key => {
    const key = keys.get(name);
    if (key == null) throw new Error(`world file references unknown channel "${name}"`);
    return key;
  };
  if (Array.isArray(value)) return value.map((v) => bind(v, keys));
  if (value == null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([k, v]) => {
      if (k === "channel" && typeof v === "string") return [k, resolve(v)];
      if (/^y[1-4]$/.test(k) && Array.isArray(v))
        return [k, v.map((n) => (typeof n === "string" ? resolve(n) : n))];
      if (k === "key" && typeof v === "string" && /^y[1-4]---/.test(v)) {
        const parts = v.split("---");
        parts[parts.length - 1] = String(resolve(parts[parts.length - 1]));
        return [k, parts.join("---")];
      }
      return [k, bind(v, keys)];
    }),
  );
};

/**
 * testStand builds an LOX coldflow stand on the capture's Core: its sensor, valve, and
 * log channels, and the TPC project (an operator schematic, a thermocouple table,
 * pressure plots, a sequence log, and the Ox Coldflow Arc program). It deploys the Arc
 * program to the Core, where the Start Sequence button runs it, and simulates the stand
 * until stopped: valves follow their commands, and tank, pressurant, and feed readings
 * follow the valves.
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

  const names = (await readdir(TPC_DIR)).filter((n) => n.endsWith(".json"));
  const files = await Promise.all(
    names.map(async (name): Promise<zip.Source> => {
      const raw = await readFile(path.join(TPC_DIR, name), "utf8");
      const bytes = new TextEncoder().encode(
        JSON.stringify(bind(JSON.parse(raw), keys)),
      );
      return { path: name, read: async () => bytes };
    }),
  );
  const bundle = new Uint8Array(await new Response(zip.create(files)).arrayBuffer());
  const project = await client.projects.import(bundle, { fileName: "TPC.zip" });

  const arc = await client.arcs.retrieve({ name: SEQUENCE });
  const [rack] = await client.racks.retrieve({ integration: "arc" });
  if (rack == null) throw new Error("the Core has no rack that runs Arc");
  const task = await client.arcs.setRack(arc.key, rack.key);
  if (task == null)
    throw new Error(`binding ${SEQUENCE} to rack ${rack.name} made no task`);
  const started = await task.executeCommandSync({
    type: "start",
    timeout: START_TIMEOUT,
  });
  if (started.variant === "error")
    throw new Error(`${SEQUENCE} failed to start: ${started.message}`);

  let time = now().sub(HISTORY);
  const writer = await client.openWriter({
    start: time,
    channels: [daq, ...SENSORS.map(key), ...VALVE_NAMES.map((v) => key(`${v}_state`))],
  });
  const logWriter = await client.openWriter({
    start: time,
    channels: [key(`${LOG}_time`), key(LOG)],
  });

  const state = Object.fromEntries(VALVE_NAMES.map((v) => [v, 0])) as Record<
    Valve,
    number
  >;
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

  const flowing = (v: Valve): boolean =>
    NORMALLY_OPEN.has(v) ? state[v] === 0 : state[v] === 1;
  const p = { ox: 0, fuel: 0, press: 180 };
  const t = { ox: -183, fuel: 19, press: 21 };

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
      const dt = SAMPLE.seconds;
      while (time.add(SAMPLE).beforeEq(target)) {
        time = time.add(SAMPLE);
        for (const [v, value] of pending) {
          if (state[v] === value) continue;
          state[v] = value;
          if (UNLOGGED.has(v)) continue;
          logStamps.push(time);
          logs.push(`${VALVES[v]} ${flowing(v) ? "open" : "closed"}`);
        }
        pending.clear();
        const feeds = Number(flowing("ox_press")) + Number(flowing("ox_press_2"));
        const inflow = flowing("press_iso")
          ? 0.25 * feeds * Math.max(0, p.press - p.ox)
          : 0;
        const vent = flowing("ox_vent") ? 0.8 * p.ox : 0;
        const flow = flowing("ox_mpv") ? 0.35 * p.ox : 0;
        const boost = flowing("gas_booster_iso")
          ? 30 * Math.max(0, 1 - p.press / 320)
          : 0;
        p.ox = Math.max(0, p.ox + (inflow - vent - flow) * dt);
        p.press = Math.max(0, p.press + (boost - 2 * inflow) * dt);
        p.fuel = flowing("fuel_vent") ? approach(p.fuel, 0, 1, dt) : p.fuel;
        t.ox = approach(t.ox, flow > 0 ? -190 : -183, 2, dt);
        t.press = approach(t.press, 21 - 0.1 * inflow, 1.5, dt);
        const sample: Record<Sensor, number> = {
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
        for (const v of VALVE_NAMES) push(key(`${v}_state`), state[v]);
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
    project: project.name,
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
