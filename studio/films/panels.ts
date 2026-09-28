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
 * An operator starts a coldflow, then moves through the project's panels while it runs:
 * one per subsystem, then one per job (automation, hardware, review), and back to
 * operations, where the plots kept every sample.
 */
export const viewport = { width: 1440, height: 900 };

export const format = "landscape";

/** The strip's panels in tour order. */
const ORDER = [
  "Operations",
  "Oxidizer",
  "Pressurant",
  "Fuel",
  "Automations",
  "Hardware",
  "Review",
];

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

export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  try {
    const { page } = session;
    await capture.login(session, world.project);
    const strip = page.locator(".console-panel-selector:visible").first();
    const pill = (name: string): Locator =>
      strip.getByText(name, { exact: true }).first();
    await session.waitFor(pill("Operations"));
    // The strip lists new panels by name, so drag them into the order the film tours.
    const pills = strip.locator("[draggable=true]");
    for (const [i, name] of ORDER.entries()) {
      const at = pills.nth(i);
      if ((await at.innerText()).trim() === name) continue;
      await pill(name).dragTo(at, { targetPosition: { x: 4, y: 8 } });
      await session.settle(300);
    }
    const order = (await pills.allInnerTexts()).map((t) => t.trim());
    if (order.join() !== ORDER.join())
      throw new Error(`the strip reads ${order.join(", ")} after arranging`);
    // A plot reads only the ranges its viewer holds, so hold the run under review.
    await session.click(pill("Review"));
    const run = page.getByText("Coldflow 11", { exact: true }).first();
    await session.waitFor(run);
    await run.click({ button: "right" });
    await page
      .locator(".pluto-menu-context")
      .getByText("Favorite", { exact: true })
      .click();
    await session.settle(500);
    const leaf = (tab: string): Locator =>
      page.locator(".pluto-mosaic__leaf:visible").filter({ hasText: tab }).first();
    const past = await box(leaf("Coldflow 11 Pressures"));

    await session.click(pill("Hardware"));
    const form = page.locator(".console-task-channel-form-container:visible").first();
    await session.waitFor(form);
    await session.settle(500);
    const daq = await union(
      page.locator(".console-task-properties:visible").first(),
      form,
    );

    await session.click(pill("Automations"));
    const line = (text: string): Locator =>
      page.locator(".view-line:visible").filter({ hasText: text }).first();
    await session.waitFor(line("func control_tpc"));
    await session.settle(500);
    const regulate = await union(line("func control_tpc"), line("ox_press_cmd = 1"));
    const sequence = { ...regulate, width: 420 };
    // The code around the highlighted function, so the push-in keeps its context.
    const code = {
      x: sequence.x - 120,
      y: sequence.y - 90,
      width: 760,
      height: 470,
    };

    await session.click(pill("Operations"));
    const diagram = page.locator(".pluto-diagram:visible").first();
    await session.waitFor(diagram);
    await session.settle(1500);
    const start = diagram.getByRole("button", { name: "Start Sequence" }).first();
    await session.click(
      page.locator(".console-controls button:has(svg.pluto-icon--circle)").first(),
    );
    await session.settle(800);
    await session.moveTo({ x: 820, y: 700 });

    const full = { x: 0, y: 0, ...viewport };
    // The strip's first pills and the top left of whichever panel is open.
    const upper = { x: 0, y: 0, width: 1000, height: 620 };
    const tab = (name: string): Locator =>
      pills.filter({ has: page.getByText(name, { exact: true }) }).first();
    const subsystems = await union(tab("Oxidizer"), tab("Pressurant"), tab("Fuel"));
    const plot = await box(leaf("OX Pressures"));
    const pressed = await box(start);
    const button = {
      x: pressed.x - 260,
      y: pressed.y - 150,
      width: 440,
      height: 340,
    };
    session.track("ox", () => world.read("ox_pt_1"));

    session.startRecording();
    await session.mark("start");
    await session.mark("subsystems", subsystems);
    await session.mark("sequence", sequence);
    await session.mark("daq", daq);
    await session.mark("past", past);
    await session.mark("plot", plot);
    await session.mark("upper", upper);
    await session.mark("code", code);
    await session.hold(400);
    await session.mark("press", button);
    await session.click(start);
    await session.mark("go");
    await session.hold(500);

    await session.mark("tour", full);
    await session.click(pill("Oxidizer"));
    await session.mark("oxidizer");
    await session.hold(1000);
    await session.click(pill("Pressurant"));
    await session.hold(700);
    await session.click(pill("Fuel"));
    await session.hold(700);
    await session.mark("split", full);

    // The camera pushes in on each panel after its click and pulls back out while the
    // cursor travels to the next pill, so every click lands in frame.
    await session.click(pill("Automations"));
    await session.mark("automations");
    await session.hold(1600);
    await session.mark("to-hardware");
    await session.click(pill("Hardware"));
    await session.mark("hardware");
    await session.hold(1700);
    await session.mark("to-review");
    await session.click(pill("Review"));
    await session.mark("review");
    await session.hold(1700);
    await session.mark("to-operations");
    await session.click(pill("Operations"));
    await session.hold(200);
    await session.mark("whole");
    await session.hold(1900);
    await session.mark("end");
  } finally {
    await world.stop();
  }
};

export const edit = film.edit([
  {
    type: "card",
    lines: ["Introducing panels.", "Complex systems. Easy to understand."],
    seconds: 2.2,
  },
  {
    type: "take",
    from: "start",
    to: "split",
    beats: [
      { at: "start", wide: true, push: 0.01 },
      { at: "press", fill: 0.5, pace: 0.9, push: 0.015 },
      { at: "tour", frame: "upper", fill: 0.9, pace: 0.8, push: 0.012 },
    ],
  },
  { type: "card", lines: ["Every workflow gets", "its own panel."], seconds: 1.4 },
  {
    type: "take",
    from: "split",
    to: "end",
    beats: [
      { at: "split", wide: true, push: 0 },
      { at: "automations", frame: "code", fill: 0.8, pace: 0.6, push: 0.012 },
      { at: "to-hardware", wide: true, pace: 0.6, push: 0 },
      { at: "hardware", frame: "daq", fill: 0.8, pace: 0.6, push: 0.012 },
      { at: "to-review", wide: true, pace: 0.6, push: 0 },
      { at: "review", frame: "past", fill: 0.8, pace: 0.6, push: 0.012 },
      { at: "to-operations", wide: true, pace: 0.6, push: 0 },
      { at: "whole", frame: "plot", fill: 0.6, pace: 0.8, push: 0.01 },
    ],
  },
  { type: "end", tagline: "One project. Every view of the system.", seconds: 2.6 },
]);

/** Settled on the whole window, before the tour moves in. */
export const thumbnail = film.thumbnail({ at: "tour", after: 0.9 });

export const overlays = film.overlays({
  scope: {
    track: "ox",
    label: "OX PT 1",
    unit: "psi",
    color: "#DC136C",
    from: "go",
    to: "end",
    zero: "go",
  },
  callouts: [
    { at: "oxidizer", target: "subsystems", text: "A panel per subsystem", seconds: 2 },
    { at: "automations", target: "sequence", text: "Write the sequence", seconds: 1.6 },
    {
      at: "hardware",
      target: "daq",
      text: "Configure the DAQ",
      seconds: 1.7,
      focused: true,
    },
    { at: "review", target: "past", text: "Review past runs", seconds: 1.7 },
    {
      at: "whole",
      target: "plot",
      text: "Continuous data across panels",
      seconds: 1.9,
    },
  ],
});
