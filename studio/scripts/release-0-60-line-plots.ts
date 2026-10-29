// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { capture, fixtures } from "@/index";

import { NEW_PANEL, renamePanel } from "./common/panels";

const RANGE = "Tank soak test";
const TITLE = "Soak test";
const PANEL = "Review";
const PRESSURE = "tank_pressure";
const RATE = 100;
const PERIOD = TimeSpan.milliseconds(1000 / RATE);
const HOURS = 24;
const SAMPLES = HOURS * 3600 * RATE;
const CHUNK = 15 * 60 * RATE;
// A fixed start, so both themes show the same axis.
const START = TimeStamp.milliseconds(new Date(2026, 9, 24, 8, 0, 0).getTime());
const END = START.add(PERIOD.mult(SAMPLES));

/** One-sample spikes as [hours into the test, psi above the trend]. */
const SPIKES: ReadonlyArray<readonly [number, number]> = [
  [0.93, 96],
  [2.61, 108],
  [4.27, 91],
  [5.48, 112],
  [7.84, 99],
  [9.12, 104],
  [11.63, 93],
  [13.37, 110],
  [15.21, 97],
  [17.72, 105],
  [19.44, 89],
  [22.08, 102],
];
const TARGET = 7;

const spikeIndex = (hours: number): number => Math.round(hours * 3600 * RATE) + 37;

/** mulberry32 returns a seeded uniform random source on [0, 1). */
const mulberry32 = (seed: number): (() => number) => {
  let a = seed;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};

/**
 * writeSoakTest writes a day of tank pressure at 100 Hz, with a slow daily swing, a
 * small leak, sensor noise, and one-sample spikes, and a range over all of it.
 */
const writeSoakTest = async (client: Synnax): Promise<void> => {
  const time = await client.channels.create({
    name: "tank_time",
    isIndex: true,
    dataType: "timestamp",
  });
  const pressure = await client.channels.create({
    name: PRESSURE,
    dataType: "float32",
    index: time.key,
  });
  const spikes = new Map(SPIKES.map(([h, psi]) => [spikeIndex(h), psi]));
  const random = mulberry32(7);
  const gaussian = (): number =>
    Math.sqrt(-2 * Math.log(1 - random())) * Math.cos(2 * Math.PI * random());
  const writer = await client.openWriter({
    start: START,
    channels: [time.key, pressure.key],
  });
  try {
    const start = BigInt(START.valueOf());
    const period = BigInt(PERIOD.valueOf());
    for (let lo = 0; lo < SAMPLES; lo += CHUNK) {
      const n = Math.min(CHUNK, SAMPLES - lo);
      const times = new BigInt64Array(n);
      const values = new Float32Array(n);
      for (let j = 0; j < n; j++) {
        const i = lo + j;
        const h = i / (3600 * RATE);
        times[j] = start + BigInt(i) * period;
        values[j] =
          503 +
          6 * Math.sin((2 * Math.PI * (h - 1)) / 24) -
          0.3 * h +
          1.6 * Math.sin((2 * Math.PI * h) / 2.7 + 0.8) +
          1.1 * gaussian() +
          (spikes.get(i) ?? 0);
      }
      await writer.write({ [time.key]: times, [pressure.key]: values });
    }
  } finally {
    await writer.close();
  }
  await client.ranges.create({ name: RANGE, timeRange: new TimeRange(START, END) });
};

/** The part of the range's time axis in view, in the plot's decimal space. */
interface View {
  left: number;
  width: number;
}

// The time axis pads the range by 1% on each side.
const X_PADDING = 0.01;

const decimal = (t: TimeStamp): number => {
  const span = Number(END.valueOf() - START.valueOf());
  const offset = Number(t.valueOf() - START.valueOf());
  return (offset + X_PADDING * span) / (span * (1 + 2 * X_PADDING));
};

// Mirrors the viewport's rounding, which keeps more digits once a side is under 0.001.
const truncate = (left: number, right: number): View => {
  const digits = Math.min(right - left, 1) < 0.001 ? 11 : 6;
  const l = Number(left.toFixed(digits));
  return { left: l, width: Number(right.toFixed(digits)) - l };
};

/**
 * Docs `releases/0-60-0/line-plots`: a day of 100 Hz tank pressure with a dozen
 * one-sample spikes. The plot opens on the whole day with every spike, zooms in four
 * drags to a second around one spike, where single samples show, and resets the zoom.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    await writeSoakTest(client);
  } finally {
    await client.close();
  }

  const { page } = session;
  await capture.login(session);
  // Opening a range from the palette makes it active, and a new plot opens on it.
  await capture.searchPalette(session, RANGE);
  await capture.clearPanel(session);
  await capture.clickPanelCreate(session);
  await capture.createComponent(session, "Line plot");
  await session.waitFor(page.locator(".pluto-line-plot").first());
  await capture.addChannels(session, "Y1", [PRESSURE], { search: "tank" });
  await capture.hideBottomToolbar(session);
  await renamePanel(session, NEW_PANEL, PANEL);

  const tab = page.locator(".console-mosaic .lyra-tabs__tab").first();
  await tab.click({ button: "right", timeout: 5000 });
  const menu = page.locator(".lyra-menu-context").first();
  await session.waitFor(menu);
  await menu
    .getByText(/^Rename/)
    .first()
    .click({ timeout: 5000 });
  await session.waitFor(page.locator(".lyra-text--editable[contenteditable='true']"));
  await page.keyboard.press("ControlOrMeta+a");
  await page.keyboard.type(TITLE);
  await page.keyboard.press("Enter");

  // The hover readout would cover the spike while the cursor rests on the plot.
  await session.click(capture.control(page, "tooltip"), { zoom: false });

  const loading = page.locator(".pluto-line-plot__loading, .pluto-line-plot__fetching");
  const settleTiles = async (): Promise<void> => {
    for (let i = 0; i < 120; i++) {
      await session.settleWall(250);
      if (
        !(await loading
          .first()
          .isVisible()
          .catch(() => false))
      )
        break;
    }
    await session.settleWall(600);
  };
  await settleTiles();
  await session.moveTo(capture.PARK);
  await session.settle(800);
  await session.settleWall(1000);

  const viewport = await page
    .locator(".pluto-line-plot__viewport")
    .first()
    .boundingBox();
  if (viewport == null) throw new Error("line plot viewport is hidden");
  const target = decimal(START.add(PERIOD.mult(spikeIndex(SPIKES[TARGET][0]))));
  let view: View = { left: 0, width: 1 };
  const screenX = (d: number): number =>
    viewport.x + ((d - view.left) / view.width) * viewport.width;
  const y = viewport.y + viewport.height * 0.3;

  // Each drag zooms the time axis around the spike by `factor`.
  const zoomIn = async (factor: number): Promise<void> => {
    const center = screenX(target);
    const half = viewport.width / factor / 2;
    const x1 = Math.round(center - half);
    const x2 = Math.round(center + half);
    await session.drag({ x: x1, y }, { x: x2, y: y + 6 }, { zoom: false });
    const at = (x: number): number =>
      view.left + ((x - viewport.x) / viewport.width) * view.width;
    view = truncate(at(x1), at(x2));
    await settleTiles();
  };

  session.startRecording();
  await session.hold(2000);

  for (const factor of [18, 20, 20, 12]) {
    await zoomIn(factor);
    await session.hold(1300);
  }
  await session.hold(900);

  await session.click(capture.control(page, "expand"), { zoom: false });
  await settleTiles();
  await session.hold(300);
  await session.moveTo(capture.PARK);
  await session.hold(2800);
};
