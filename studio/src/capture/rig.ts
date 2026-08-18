// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

import {
  type Browser,
  type CDPSession,
  chromium,
  type Locator,
  type Page,
} from "playwright";

import {
  TRAVEL_MAX_S,
  TRAVEL_MIN_S,
  TRAVEL_SCALE_S,
} from "@/director/constants";
import { type Event, type Meta, type Point, type Timeline } from "@/timeline";

export interface CaptureOptions {
  /** URL of the Console web build. */
  url: string;
  /** Directory receiving frames/ and timeline.json. */
  outDir: string;
  width?: number;
  height?: number;
  dsf?: number;
  fps?: number;
  theme?: "light" | "dark";
  headed?: boolean;
}

/**
 * Pauses any newly discovered Web Animations (CSS transitions/animations, WAAPI)
 * and advances them manually per tick, so compositor-clocked motion steps in
 * lockstep with the captured virtual clock instead of wall time.
 */
const ANIMATION_STEPPER = `(() => {
  const elapsed = new WeakMap();
  window.__studioStepAnimations = (dt) => {
    for (const anim of document.getAnimations()) {
      try {
        if (anim.playState === "finished" || anim.playState === "idle") continue;
        if (!elapsed.has(anim)) {
          anim.pause();
          elapsed.set(anim, Number(anim.currentTime ?? 0));
        }
        const t = elapsed.get(anim) + dt;
        elapsed.set(anim, t);
        const timing = anim.effect?.getComputedTiming();
        const end = timing == null ? null : Number(timing.endTime);
        if (end != null && Number.isFinite(end) && t >= end) anim.finish();
        else anim.currentTime = t;
      } catch {}
    }
  };
})();`;

/**
 * CaptureSession drives the Console under a stepped virtual clock, saving one
 * lossless PNG per tick while recording and logging every synthetic input event
 * to the timeline. Actions dispatch real Playwright input; the on-screen cursor
 * is synthesized later by the director, so nothing cursor-shaped renders here.
 */
export class CaptureSession {
  private readonly browser: Browser;
  readonly page: Page;
  private readonly cdp: CDPSession;
  private readonly opts: Required<CaptureOptions>;
  private readonly events: Event[] = [];
  private frame = 0;
  private recording = false;
  private cursor: Point;
  private origin: Point | null = null;

  private constructor(
    browser: Browser,
    page: Page,
    cdp: CDPSession,
    opts: Required<CaptureOptions>,
  ) {
    this.browser = browser;
    this.page = page;
    this.cdp = cdp;
    this.opts = opts;
    this.cursor = { x: opts.width / 2, y: opts.height / 2 };
  }

  static async launch(options: CaptureOptions): Promise<CaptureSession> {
    const opts: Required<CaptureOptions> = {
      width: 1920,
      height: 1080,
      dsf: 2,
      fps: 60,
      theme: "light",
      headed: false,
      ...options,
    };
    const browser = await chromium.launch({
      headless: !opts.headed,
      args: ["--force-color-profile=srgb", "--hide-scrollbars"],
    });
    const context = await browser.newContext({
      viewport: { width: opts.width, height: opts.height },
      deviceScaleFactor: opts.dsf,
      colorScheme: opts.theme,
    });
    const page = await context.newPage();
    await page.clock.install();
    await page.addInitScript(ANIMATION_STEPPER);
    await page.goto(opts.url, { timeout: 30_000 });
    const cdp = await context.newCDPSession(page);
    await mkdir(path.join(opts.outDir, "frames"), { recursive: true });
    return new CaptureSession(browser, page, cdp, opts);
  }

  private get tickMs(): number {
    return 1000 / this.opts.fps;
  }

  /** tick advances the virtual clock one frame and captures it if recording. */
  async tick(): Promise<void> {
    await this.page.clock.runFor(this.tickMs);
    await this.page.evaluate(
      (dt) => (window as any).__studioStepAnimations?.(dt),
      this.tickMs,
    );
    if (!this.recording) return;
    const { data } = await this.cdp.send("Page.captureScreenshot", {
      format: "png",
      fromSurface: true,
      optimizeForSpeed: true,
      captureBeyondViewport: false,
    });
    const name = String(this.frame).padStart(6, "0");
    await writeFile(
      path.join(this.opts.outDir, "frames", `${name}.png`),
      Buffer.from(data, "base64"),
    );
    this.frame++;
  }

  /** settle advances virtual time without capturing; use during setup. */
  async settle(ms: number): Promise<void> {
    const ticks = Math.ceil(ms / this.tickMs);
    for (let i = 0; i < ticks; i++) {
      await this.page.clock.runFor(this.tickMs);
      await this.page.evaluate(
        (dt) => (window as any).__studioStepAnimations?.(dt),
        this.tickMs,
      );
    }
  }

  /** startRecording begins frame capture; the cursor origin is its position now. */
  startRecording(): void {
    this.recording = true;
    this.origin = { ...this.cursor };
  }

  /** hold captures ms of unchanged (but still ticking) screen time. */
  async hold(ms: number): Promise<void> {
    const ticks = Math.round(ms / this.tickMs);
    for (let i = 0; i < ticks; i++) await this.tick();
  }

  private async resolve(target: Locator | Point): Promise<Point> {
    if ("x" in target && typeof target.x === "number") return target;
    const locator = target as Locator;
    const box = await locator.boundingBox();
    if (box == null) throw new Error("capture target has no bounding box");
    return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
  }

  private travelTicks(to: Point): number {
    const { width, height, fps } = this.opts;
    const d = Math.hypot(to.x - this.cursor.x, to.y - this.cursor.y);
    const diag = Math.hypot(width, height);
    const s = Math.min(
      TRAVEL_MAX_S,
      TRAVEL_MIN_S + TRAVEL_SCALE_S * Math.sqrt(d / diag),
    );
    return Math.round(s * fps);
  }

  /**
   * moveTo travels the virtual cursor to the target: the travel time elapses on
   * screen first, then the real mouse move dispatches so hover state appears at
   * arrival, matching the synthesized cursor's arrival in post.
   */
  async moveTo(target: Locator | Point): Promise<Point> {
    const to = await this.resolve(target);
    const duration = this.travelTicks(to);
    for (let i = 0; i < duration; i++) await this.tick();
    this.events.push({ type: "move", tick: this.frame, ...to, duration });
    await this.page.mouse.move(to.x, to.y);
    this.cursor = to;
    await this.tick();
    return to;
  }

  /** click travels to the target, presses, and releases. */
  async click(target: Locator | Point): Promise<void> {
    const at = await this.moveTo(target);
    this.events.push({ type: "pointerdown", tick: this.frame, ...at, button: "left" });
    await this.page.mouse.down();
    await this.hold(80);
    this.events.push({ type: "pointerup", tick: this.frame, ...at, button: "left" });
    await this.page.mouse.up();
    await this.tick();
  }

  /** type enters text at a human cadence, one key per interval. */
  async type(text: string, msPerChar = 55): Promise<void> {
    for (const char of text) {
      this.events.push({ type: "key", tick: this.frame, key: char });
      await this.page.keyboard.type(char);
      await this.hold(msPerChar);
    }
  }

  /** press dispatches a single named key (e.g. "Enter", "Escape"). */
  async press(key: string): Promise<void> {
    this.events.push({ type: "key", tick: this.frame, key });
    await this.page.keyboard.press(key);
    await this.tick();
  }

  /**
   * waitFor polls a locator while advancing virtual time, so app timers keep
   * firing while the rig waits for the UI to settle.
   */
  async waitFor(locator: Locator, timeoutTicks = 600): Promise<void> {
    for (let i = 0; i < timeoutTicks; i++) {
      if (await locator.isVisible().catch(() => false)) return;
      await this.tick();
    }
    throw new Error(`timed out waiting for ${String(locator)}`);
  }

  /** finish writes the timeline and closes the browser, returning the timeline. */
  async finish(): Promise<Timeline> {
    const meta: Meta = {
      version: 1,
      fps: this.opts.fps,
      width: this.opts.width,
      height: this.opts.height,
      dsf: this.opts.dsf,
      theme: this.opts.theme,
      frames: Math.max(1, this.frame),
    };
    const timeline: Timeline = {
      meta,
      events: this.events,
      origin: this.origin ?? this.cursor,
    };
    await writeFile(
      path.join(this.opts.outDir, "timeline.json"),
      JSON.stringify(timeline, null, 2),
    );
    await this.browser.close();
    return timeline;
  }
}
