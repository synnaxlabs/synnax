// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, film, fixtures } from "@/index";
import { type Rect } from "@/timeline";

/**
 * An operator presses Start Sequence and the stand's Arc program runs an LOX
 * coldflow: it pressurizes the tank, boosts the pressurant, flows through the main
 * valve while cycling the TPC valve, and safes the stand.
 */
export const viewport = { width: 1440, height: 900 };

const box = async (locator: Locator): Promise<Rect> => {
  const b = await locator.boundingBox();
  if (b == null) throw new Error(`${String(locator)} is not on screen`);
  return b;
};

/** union returns the smallest rect that holds every locator's bounding box. */
const union = async (...locators: Locator[]): Promise<Rect> => {
  const boxes = await Promise.all(locators.map(box));
  const left = Math.min(...boxes.map((b) => b.x));
  const top = Math.min(...boxes.map((b) => b.y));
  const right = Math.max(...boxes.map((b) => b.x + b.width));
  const bottom = Math.max(...boxes.map((b) => b.y + b.height));
  return { x: left, y: top, width: right - left, height: bottom - top };
};

/** around returns a rect of the given size centered on a locator or rect. */
const around = async (
  target: Locator | Rect,
  width: number,
  height: number,
): Promise<Rect> => {
  const b = "x" in target ? target : await box(target);
  const cx = b.x + b.width / 2;
  const cy = b.y + b.height / 2;
  return { x: cx - width / 2, y: cy - height / 2, width, height };
};

export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  try {
    const { page } = session;
    await capture.login(session, world.project);
    // The hidden Automations panel stays mounted, so match visible nodes only.
    await session.click(page.getByText("Primary", { exact: true }).first());
    const diagram = page.locator(".pluto-diagram:visible").first();
    await session.waitFor(diagram);
    await session.settle(1500);
    const leaf = (tab: string): Locator =>
      page.locator(".pluto-mosaic__leaf:visible").filter({ hasText: tab }).first();
    const label = (text: string): Locator =>
      diagram.getByText(text, { exact: true }).first();
    // The Log draws on a canvas, so the stand reports its lines instead of the DOM.
    const logged = async (line: string): Promise<void> =>
      await session.waitFor(() => world.logged.includes(line));
    const start = diagram.getByRole("button", { name: "Start Sequence" }).first();

    await session.click(
      page.locator(".console-controls button:has(svg.pluto-icon--circle)").first(),
    );
    await session.settle(800);
    await session.moveTo({ x: 820, y: 700 });
    const pressed = await box(start);
    // The plots border the button on the right, so the frame leans onto the engine.
    const button = await around({ ...pressed, x: pressed.x - 90 }, 340, 260);
    const rest = { x: pressed.x + pressed.width + 60, y: pressed.y + 70 };
    const tank = await around(label("OX TC 1"), 300, 300);
    const valves = await union(label("OX TPC 1"), label("OX TPC 2"));
    const valve = await around(valves, 300, 220);
    const plot = await box(leaf("OX Pressures"));
    const recent = { ...plot, x: plot.x + plot.width * 0.45, width: plot.width * 0.55 };
    const logBox = await box(leaf("Sequence Logs"));
    const lines = { ...logBox, width: 400, height: 190 };
    const iso = await box(label("OX Press Iso"));
    const readout = await box(
      diagram
        .locator(".react-flow__node")
        .filter({ has: page.getByText("OX PT 1", { exact: true }) })
        .first(),
    );
    session.track("ox", () => world.read("ox_pt_1"));

    session.startRecording();
    await session.mark("start", button);
    await session.mark("iso", iso);
    await session.mark("tpc", valves);
    await session.mark("readout", readout);
    await session.hold(900);
    await session.click(start);
    await session.mark("go");
    await session.hold(150);
    await session.mark("pressing", tank);
    // The hand comes off the button and rests beside it.
    await session.moveTo({ x: rest.x, y: rest.y });
    await session.mark("pressurizing");
    await logged("OX Press Iso closed");
    await session.hold(1400);
    await session.mark("boosting", plot);
    await logged("OX MPV open");
    await session.hold(1500);
    await session.mark("sawtooth", recent);
    await session.hold(3800);
    await session.mark("regulating", valve);
    await session.hold(800);
    await session.mark("holding");
    await session.hold(2700);
    await session.mark("regulated");
    await logged("OX MPV closed");
    await session.hold(200);
    await session.mark("safing", lines);
    await session.hold(2000);
    await session.mark("venting", tank);
    await session.hold(400);
    await session.mark("safed");
    await session.hold(1100);
    await session.mark("stand");
    await session.hold(3000);
    await session.mark("end");
  } finally {
    await world.stop();
  }
};

export const edit = film.edit([
  { type: "card", lines: ["Press start.", "Arc runs the coldflow."] },
  {
    type: "take",
    from: "start",
    to: "boosting",
    beats: [
      { at: "start", fill: 0.62, push: 0.02 },
      { at: "pressing", fill: 0.62, pace: 1.3, push: 0.015 },
    ],
  },
  {
    type: "take",
    from: "boosting",
    to: "regulating",
    beats: [
      { at: "boosting", fill: 0.9, push: 0 },
      { at: "sawtooth", fill: 0.7, pace: 1.6 },
    ],
  },
  {
    type: "take",
    from: "regulating",
    to: "regulated",
    beats: [{ at: "regulating", fill: 0.62, push: 0.012 }],
  },
  {
    type: "take",
    from: "safing",
    to: "venting",
    beats: [{ at: "safing", fill: 0.72, push: 0.015 }],
  },
  {
    type: "take",
    from: "venting",
    to: "end",
    beats: [
      { at: "venting", fill: 0.62, push: 0 },
      { at: "stand", wide: true, pace: 1.9, push: 0.006 },
    ],
  },
  { type: "end", tagline: "Run the test. Watch every channel." },
]);

export const overlays = film.overlays({
  scope: {
    track: "ox",
    label: "OX PT 1",
    unit: "psi",
    color: "#DC136C",
    from: "start",
    to: "stand",
    zero: "go",
  },
  callouts: [
    { at: "pressurizing", target: "iso", text: "Arc pressurizes to 50 psi" },
    { at: "holding", target: "tpc", text: "Arc holds 20 to 25 psi" },
    { at: "safed", target: "readout", text: "Arc vents under 15 psi" },
  ],
});
