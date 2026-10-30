// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, log, type Synnax } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { isolateTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

/**
 * A threshold alarm on a pressure sensor. It logs when the reading crosses `trip` and
 * again when it returns past `clear`. It arms only once the reading first sits on the
 * normal side of `clear`, so the low alarms stay quiet until the run pressurizes.
 */
interface Alarm {
  sensor: string;
  label: string;
  low: boolean;
  trip: number;
  clear: number;
}

/** Limits the coldflow crosses: the TPC band of the tank and the pressurant dips. */
const ALARMS: Alarm[] = [
  { sensor: "ox_pt_1", label: "OX PT 1", low: false, trip: 24.5, clear: 23 },
  { sensor: "ox_pt_1", label: "OX PT 1", low: true, trip: 20.5, clear: 22 },
  { sensor: "press_pt_1", label: "Press PT 1", low: false, trip: 140, clear: 130 },
  { sensor: "press_pt_1", label: "Press PT 1", low: true, trip: 115, clear: 120 },
];

/**
 * startAlarms creates the `alarm_log` channel and writes a line to it each time a live
 * reading crosses the limits of an alarm. Returns the channel's key and a function that
 * stops the alarms.
 */
const startAlarms = async (
  client: Synnax,
): Promise<{ key: channel.Key; stop: () => Promise<void> }> => {
  const index = await client.channels.create({
    name: "alarm_log_time",
    isIndex: true,
    dataType: "timestamp",
  });
  const alarmLog = await client.channels.create({
    name: "alarm_log",
    dataType: "string",
    index: index.key,
  });
  const sensors = await client.channels.retrieve([
    ...new Set(ALARMS.map((a) => a.sensor)),
  ]);
  const sensor = (name: string): channel.Key => {
    const ch = sensors.find((s) => s.name === name);
    if (ch == null) throw new Error(`the stand has no sensor "${name}"`);
    return ch.key;
  };
  const writer = await client.openWriter({ channels: [alarmLog.key], autoIndex: true });
  const streamer = await client.openStreamer(sensors.map((s) => s.key));
  const states = ALARMS.map((): "idle" | "normal" | "tripped" => "idle");
  const loop = (async () => {
    for await (const frame of streamer) {
      const lines: string[] = [];
      ALARMS.forEach((a, i) => {
        for (const sample of frame.get(sensor(a.sensor))) {
          const v = Number(sample);
          const normal = a.low ? v > a.clear : v < a.clear;
          if (states[i] !== "tripped" && normal) states[i] = "normal";
          else if (states[i] === "normal" && (a.low ? v < a.trip : v > a.trip)) {
            states[i] = "tripped";
            lines.push(`${a.label} ${a.low ? "low" : "high"}: ${v.toFixed(1)} psi`);
          } else if (states[i] === "tripped" && normal) {
            states[i] = "normal";
            lines.push(`${a.label} normal: ${v.toFixed(1)} psi`);
          }
        }
      });
      if (lines.length > 0) await writer.write(alarmLog.key, lines);
    }
  })();
  return {
    key: alarmLog.key,
    stop: async () => {
      streamer.close();
      await loop;
      await writer.close();
    },
  };
};

/**
 * Still for the hero of docs page `reference/visualize/logs`: the TPC stand's Sequence
 * Logs alone in the Operations panel after a coldflow, with its toolbar open. The log
 * mixes each valve change of the sequence with alarms on live tank and pressurant
 * readings.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  let stopAlarms: (() => Promise<void>) | undefined;
  try {
    const { page } = session;
    const alarms = await startAlarms(client);
    stopAlarms = alarms.stop;
    // Set before the log first mounts: a log that changes channels while open skips
    // the first frame of each one.
    const logs = await client.ontology.retrieve({ types: ["log"] });
    const resource = logs.find((r) => r.name === "Sequence Logs");
    if (resource == null) throw new Error("the TPC world has no Sequence Logs log");
    const { key } = await client.logs.retrieve({ key: resource.id.key });
    const sequence = await client.channels.retrieve("sequence_log");
    const dispatched = await client.logs.dispatch(key, [
      log.setChannelAlias({ channel: sequence.key, alias: "Sequence" }),
      log.addChannel({ channel: alarms.key }),
      log.setChannelAlias({ channel: alarms.key, alias: "Alarms" }),
      log.setChannelColor({ channel: alarms.key, color: "#d97706" }),
      log.setTimestampPrecision({ timestampPrecision: 3 }),
      log.setChannelNamesHidden({ channelNamesHidden: false }),
    ]);
    if (!dispatched) throw new Error("the Sequence Logs log is not cached");

    await capture.login(session, world.project);
    await session.click(page.getByText("Operations", { exact: true }).first());
    const diagram = page.locator(".pluto-diagram:visible").first();
    await session.waitFor(diagram);
    await session.settle(1500);
    // A log shows only values that arrive while it is open, so it stays in the panel
    // through the whole coldflow.
    await session.click(page.getByText("Operator", { exact: true }).first());
    await session.settle(500);
    await session.click(capture.control(page, "circle"));
    await session.settle(800);
    await session.click(
      diagram.getByRole("button", { name: "Start Sequence" }).first(),
    );
    await session.waitFor(() => world.logged.includes("OX Vent open"), 3600);
    await session.settle(1500);

    await isolateTab(session, "Sequence Logs");
    await session.click(
      page
        .locator(".console-mosaic .lyra-tabs__tab")
        .getByText("Sequence Logs", { exact: true })
        .first(),
      { zoom: false },
    );
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
    const drawer = page.locator(".console-nav__drawer.lyra--location-bottom").first();
    await session.waitFor(drawer.getByText("Add channel", { exact: true }).first());
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    // The log redraws at its new size on wall time, which virtual ticks outrun.
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await stopAlarms?.();
    await client.close();
    await world.stop();
  }
};
