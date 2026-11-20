// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ranger,
  type Synnax,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";
import type { Locator } from "playwright";

import { capture, fixtures } from "@/index";

import { dropTemporal, pinClock } from "./common/clock";
import { isolateTab } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

const CAMPAIGN = "LOX coldflow campaign";
const RUNS = ["Coldflow 9", "Coldflow 10", "Coldflow 11"];
const RUN_COLOR = "#7C5CE0";
const PLOT = "OX Pressures";
/** Video time the flow phase takes, however long the stand flows. */
const FLOW_SHOWN = TimeSpan.seconds(4);
const REST = { x: 26, y: 600 };
/**
 * How far the virtual clock runs ahead of wall time before the capture. Arc runs on
 * wall time and the stand on the virtual clock, which falls behind wall time while
 * frames record. The stand's samples must stay ahead for the sequence to advance.
 */
const LEAD = TimeSpan.minutes(30);

/** changes returns the times at which a valve on the stand changes state during tr. */
const changes = async (
  client: Synnax,
  tr: TimeRange,
  valve: string,
): Promise<TimeStamp[]> => {
  const times = Array.from(await client.read(tr, "daq_time"));
  const states = Array.from(await client.read(tr, `${valve}_state`));
  return states.flatMap((s, i) =>
    i > 0 && s !== states[i - 1] ? [new TimeStamp(times[i] as bigint)] : [],
  );
};

/**
 * Video for the ranges section of the v0.62 release notes: during a live coldflow, an
 * engineer starts Coldflow 12 under the running campaign from the Ranges toolbar,
 * starts its Pressurization phase, and clicks Next into Flow and Safing as the stand
 * moves through them. Each name comes from Coldflow 11, and the plot's range strip
 * shows each phase as it runs. Stopping Coldflow 12 moves it to Recent.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  session.setSpeed(60);
  while (session.now().before(TimeStamp.now().add(LEAD))) await session.tick();
  session.setSpeed(1);
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const runs = await Promise.all(RUNS.map((n) => client.ranges.retrieve(n)));
    const campaign = await client.ranges.create({
      name: CAMPAIGN,
      color: "#3B7DDB",
      timeRange: new TimeRange(
        runs[0].timeRange.start.sub(TimeSpan.minutes(12)),
        TimeStamp.MAX,
      ),
    });
    for (const run of runs)
      await client.ranges.dispatch(run.key, [
        ranger.set({ path: ["parent"], value: campaign.key }),
        ranger.set({ path: ["color"], value: RUN_COLOR }),
      ]);
    // The phases of the last run, which Next offers again in the same order.
    const last = runs[runs.length - 1];
    const tr = last.timeRange;
    const [closed] = await changes(client, tr, "ox_vent");
    const [opened, shut] = await changes(client, tr, "ox_mpv");
    const phase = (name: string, color: string, start: TimeStamp, end: TimeStamp) => ({
      name,
      color,
      timeRange: new TimeRange(start, end),
      parent: last.key,
    });
    await client.ranges.create([
      phase("Pressurization", "#3774D0", closed, opened),
      phase("Flow", "#F5C400", opened, shut),
      phase("Safing", "#10B061", shut, tr.end),
    ]);
    const flowSpeed = Math.max(
      1,
      new TimeRange(opened, shut).span.seconds / FLOW_SHOWN.seconds,
    );

    await pinClock(session);
    await capture.login(session, world.project);
    await session.click(page.getByText("Oxidizer", { exact: true }).first());
    const plot = page.locator(".pluto-line-plot:visible").first();
    await session.waitFor(plot);
    await isolateTab(session, PLOT);
    await dropTemporal(page);
    await capture.openToolbar(session, "range");
    await capture.resizeToolbar(session, 320);
    const toolbar = page.locator(".console-nav__drawer.lyra--location-left").first();
    const running = toolbar.locator('section[aria-label="Running"]');
    const recent = toolbar.locator('section[aria-label="Recent"]');
    const line = (name: string): Locator =>
      toolbar.getByRole("listitem", { name, exact: true }).first();
    await session.waitFor(line(CAMPAIGN));
    await session.moveTo(REST);
    await session.settleWall(1500);
    await session.settle(1200);

    const box = await plot.boundingBox();
    if (box == null) throw new Error("the OX Pressures plot is not on screen");
    // The top of the plot: the range strip over the pressures.
    const stripZoom = 1.45;
    const strip = {
      x: box.x + box.width / 2,
      y: box.y + viewport.height / stripZoom / 2 - 24,
    };
    const showToolbar = async (): Promise<void> => {
      session.endZoom();
      await session.zoom(running, 2);
    };
    const showStrip = async (): Promise<void> => {
      session.endZoom();
      await session.moveTo(REST);
      await session.zoom(strip, stripZoom);
    };
    const action = async (name: string, tooltip: string): Promise<void> => {
      const button = line(name)
        .getByRole("button", { name: tooltip, exact: true })
        .first();
      await session.moveTo(button);
      await session.hold(300);
      await session.click(button, { zoom: false });
    };
    const accept = async (name: string): Promise<void> => {
      await session.waitFor(toolbar.locator(`input[value="${name}"]`));
      await session.hold(700);
      await session.press("Enter");
      await session.waitFor(line(name));
    };

    session.startRecording();
    await session.hold(800);
    await showToolbar();
    await action(CAMPAIGN, "Start child range");
    await accept("Coldflow 12");
    // The operator starts the sequence, as the schematic's Start Sequence button does.
    // The command carries the virtual time, so Arc sees the stand's samples as newer.
    const start = session.now();
    const writer = await client.openWriter({
      start,
      channels: ["start_sim_cmd_time", "start_sim_cmd"],
    });
    await writer.write({ start_sim_cmd_time: start, start_sim_cmd: 1 });
    await writer.close();
    await action("Coldflow 12", "Start child range");
    await accept("Pressurization");
    await showStrip();
    await session.waitFor(() => world.logged.includes("OX MPV open"), 3600);
    await showToolbar();
    await action("Pressurization", "Stop and start next");
    await accept("Flow");
    await showStrip();
    session.setSpeed(flowSpeed);
    await session.waitFor(() => world.logged.includes("OX MPV closed"), 7200);
    session.setSpeed(1);
    await showToolbar();
    await action("Flow", "Stop and start next");
    await accept("Safing");
    await showStrip();
    await session.hold(3000);
    await showToolbar();
    // Stop ends only its own range, so the running phase stops first.
    await action("Safing", "Stop");
    await session.waitForHidden(running.getByRole("listitem", { name: "Safing" }));
    await session.hold(300);
    await action("Coldflow 12", "Stop");
    await session.waitFor(recent.getByRole("listitem", { name: "Coldflow 12" }));
    await session.moveTo(REST);
    await session.zoom([running, recent.getByRole("listitem").first()], 2);
    await session.hold(2000);
    session.endZoom();
    await session.hold(1500);
  } finally {
    await client.close();
    await world.stop();
  }
};
