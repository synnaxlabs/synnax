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

import { TRAVEL_MAX_S, TRAVEL_MIN_S, TRAVEL_SCALE_S } from "@/director/constants";
import { minimumJerk } from "@/director/cursor";
import {
  type CursorKind,
  type Event,
  type Meta,
  type Point,
  type Rect,
  type Timeline,
} from "@/timeline";

/**
 * Fixed virtual-clock epoch for every capture. A constant epoch keeps captures
 * reproducible run to run and gives both videos of a themed pair identical
 * on-screen timestamps (plot axes, range pickers).
 */
const CLOCK_EPOCH = new Date("2026-08-18T09:00:00");

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
  /**
   * Hides the text caret. The native caret blinks on renderer wall time, which
   * capture cannot step, so it blinks fps/wall-rate times too fast in output.
   */
  hideCaret?: boolean;
  /**
   * Hides the notification feed (default true): environment noise like stale
   * driver warnings must not pollute shots. Disable for scripts that
   * demonstrate notifications.
   */
  hideNotifications?: boolean;
  /**
   * Port of the core the capture runs against. Injected into the dev Console
   * via its dev-connection port override before the app boots.
   */
  corePort?: number;
}

/** Mirrors DEV_PORT_KEY in console/src/cluster/detectConnection.ts. */
const DEV_PORT_KEY = "synnax-dev-connection-port";

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
  private speed = 1;
  private cursor: Point;
  private cursorRect: Rect | undefined;
  private origin: Point | null = null;
  private openZoom: {
    tick: number;
    point: Point;
    rect?: Rect;
    amount?: number;
  } | null = null;

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
      width: 1512,
      height: 945,
      dsf: 2,
      fps: 60,
      theme: "light",
      headed: false,
      hideCaret: false,
      hideNotifications: true,
      corePort: 9090,
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
    await page.clock.install({ time: CLOCK_EPOCH });
    await page.addInitScript(ANIMATION_STEPPER);
    await page.addInitScript(
      `localStorage.setItem(${JSON.stringify(DEV_PORT_KEY)}, ${JSON.stringify(String(opts.corePort))});`,
    );
    const hideRules = [
      ...(opts.hideCaret ? ["* { caret-color: transparent !important; }"] : []),
      ...(opts.hideNotifications
        ? [".pluto-notification, .console-notifications { display: none !important; }"]
        : []),
    ];
    if (hideRules.length > 0)
      await page.addInitScript(`document.addEventListener("DOMContentLoaded", () => {
        const style = document.createElement("style");
        style.textContent = ${JSON.stringify(hideRules.join("\n"))};
        document.head.appendChild(style);
      });`);
    await page.goto(opts.url, { timeout: 30_000 });
    const cdp = await context.newCDPSession(page);
    await mkdir(path.join(opts.outDir, "frames"), { recursive: true });
    return new CaptureSession(browser, page, cdp, opts);
  }

  private get tickMs(): number {
    return 1000 / this.opts.fps;
  }

  /** App-time milliseconds represented by one captured output frame. */
  private get stepMs(): number {
    return this.tickMs * this.speed;
  }

  /**
   * setSpeed changes the presentation speed of subsequent capture: each output
   * frame represents `factor` frames of app time, so factor > 1 fast-forwards
   * (e.g. speed through typing) and factor < 1 is smooth slow motion (each
   * captured frame samples a smaller clock step; no interpolation involved).
   * Durations passed to hold/type remain in app time, so a hold(1000) at speed
   * 2 occupies half a second of video.
   */
  setSpeed(factor: number): void {
    if (!(factor > 0)) throw new Error(`speed must be positive, got ${factor}`);
    this.speed = factor;
  }

  /** tick advances the virtual clock one frame and captures it if recording. */
  async tick(): Promise<void> {
    await this.page.clock.runFor(this.stepMs);
    await this.page.evaluate(
      (dt) => (window as any).__studioStepAnimations?.(dt),
      this.stepMs,
    );
    if (!this.recording) return;
    // A plain surface capture returns CSS-pixel resolution even when the context
    // emulates a higher device scale factor; the clip's scale forces rendering at
    // full device resolution (width*dsf x height*dsf).
    const { data } = await this.cdp.send("Page.captureScreenshot", {
      format: "png",
      fromSurface: true,
      optimizeForSpeed: true,
      captureBeyondViewport: false,
      clip: {
        x: 0,
        y: 0,
        width: this.opts.width,
        height: this.opts.height,
        scale: this.opts.dsf,
      },
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

  /**
   * settleWall waits real time without advancing the virtual clock. Live
   * telemetry streams and the Aether render loop run on wall time, so setup
   * uses this to let a plot's stream buffer fill before recording starts.
   */
  async settleWall(ms: number): Promise<void> {
    await new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * startRecording begins frame capture: the cursor origin is its position now
   * and input events from setup are dropped, so scripts can stage the shot with
   * the same helpers they record with.
   */
  startRecording(): void {
    this.recording = true;
    this.origin = { ...this.cursor };
    this.events.length = 0;
  }

  /** hold captures ms of app time (scaled to video time by the current speed). */
  async hold(ms: number): Promise<void> {
    const ticks = Math.round(ms / this.stepMs);
    for (let i = 0; i < ticks; i++) await this.tick();
  }

  private async resolve(
    target: Locator | Point,
  ): Promise<{ point: Point; rect?: Rect; cursor?: CursorKind }> {
    if ("x" in target && typeof target.x === "number") return { point: target };
    const locator = target as Locator;
    const box = await locator.boundingBox();
    if (box == null) throw new Error("capture target has no bounding box");
    const style = await locator
      .evaluate((el) => getComputedStyle(el).cursor)
      .catch(() => "default");
    const cursor: CursorKind =
      style === "pointer" ? "pointer" : style === "text" ? "text" : "default";
    return {
      point: { x: box.x + box.width / 2, y: box.y + box.height / 2 },
      rect: box,
      cursor,
    };
  }

  private travelTicks(to: Point): number {
    const { width, height, fps } = this.opts;
    const d = Math.hypot(to.x - this.cursor.x, to.y - this.cursor.y);
    const diag = Math.hypot(width, height);
    const s = Math.min(
      TRAVEL_MAX_S,
      TRAVEL_MIN_S + TRAVEL_SCALE_S * Math.sqrt(d / diag),
    );
    return Math.round((s * fps) / this.speed);
  }

  /**
   * moveTo travels the virtual cursor to the target: the travel time elapses on
   * screen first, then the real mouse move dispatches so hover state appears at
   * arrival, matching the synthesized cursor's arrival in post.
   */
  async moveTo(target: Locator | Point): Promise<Point> {
    let { point: to, rect, cursor } = await this.resolve(target);
    const duration = this.travelTicks(to);
    for (let i = 0; i < duration; i++) await this.tick();
    // Layout can shift while travel ticks elapse (drawer/dialog animations), so
    // re-resolve locators right before input dispatch; raw mouse events land on
    // coordinates, not elements, and stale ones hit the wrong target.
    if (!("x" in target && typeof target.x === "number"))
      ({ point: to, rect, cursor } = await this.resolve(target));
    this.events.push({ type: "move", tick: this.frame, ...to, duration, rect, cursor });
    await this.page.mouse.move(to.x, to.y);
    this.cursor = to;
    this.cursorRect = rect;
    await this.tick();
    return to;
  }

  private async pressRelease(at: Point, button: "left" | "right"): Promise<void> {
    this.events.push({
      type: "pointerdown",
      tick: this.frame,
      ...at,
      button,
      rect: this.cursorRect,
    });
    await this.page.mouse.down({ button });
    await this.hold(80);
    this.events.push({ type: "pointerup", tick: this.frame, ...at, button });
    await this.page.mouse.up({ button });
    await this.tick();
  }

  /** click travels to the target, presses, and releases. */
  async click(target: Locator | Point): Promise<void> {
    await this.pressRelease(await this.moveTo(target), "left");
  }

  /** rightClick travels to the target and opens its context menu. */
  async rightClick(target: Locator | Point): Promise<void> {
    await this.pressRelease(await this.moveTo(target), "right");
  }

  /**
   * drag presses at `from`, travels to `to` with the button held, and releases.
   * Unlike moveTo, the real mouse tracks the travel tick by tick (along the same
   * minimum-jerk profile the director renders), so drag interactions follow the
   * cursor on screen.
   */
  async drag(from: Locator | Point, to: Locator | Point): Promise<void> {
    const start = await this.moveTo(from);
    this.events.push({
      type: "pointerdown",
      tick: this.frame,
      ...start,
      button: "left",
      rect: this.cursorRect,
    });
    await this.page.mouse.down();
    await this.hold(120);

    const { point: dest, rect, cursor } = await this.resolve(to);
    const duration = this.travelTicks(dest);
    for (let i = 1; i <= duration; i++) {
      const s = minimumJerk(i / duration);
      await this.page.mouse.move(
        start.x + (dest.x - start.x) * s,
        start.y + (dest.y - start.y) * s,
      );
      await this.tick();
    }
    this.events.push({
      type: "move",
      tick: this.frame,
      ...dest,
      duration,
      rect,
      cursor,
    });
    this.cursor = dest;
    this.cursorRect = rect;
    await this.hold(120);

    this.events.push({ type: "pointerup", tick: this.frame, ...dest, button: "left" });
    await this.page.mouse.up();
    await this.tick();
  }

  /**
   * zoom opens an authored camera override framing the target, superseding
   * auto-zoom until endZoom (or finish) closes it. When amount is omitted the
   * director derives it from the target's rect: as tight as the framing margin
   * allows, capped at RECT_ZOOM_MAX.
   */
  async zoom(target: Locator | Point, amount?: number): Promise<void> {
    if (this.openZoom != null) this.endZoom();
    const { point, rect } = await this.resolve(target);
    if (amount == null && rect == null)
      throw new Error("zoom on a bare point requires an explicit amount");
    this.openZoom = { tick: this.frame, point, rect, amount };
  }

  /** endZoom closes the open authored zoom, returning the camera to auto. */
  endZoom(): void {
    if (this.openZoom == null) return;
    const { tick, point, rect, amount } = this.openZoom;
    this.openZoom = null;
    if (this.frame <= tick) return;
    this.events.push({
      type: "zoom",
      tick,
      endTick: this.frame,
      amount,
      ...point,
      rect,
    });
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
    const shot = path.join(this.opts.outDir, "waitfor-timeout.png");
    await this.page.screenshot({ path: shot }).catch(() => {});
    throw new Error(`timed out waiting for ${String(locator)}; state in ${shot}`);
  }

  /** waitForHidden polls until the locator disappears, advancing virtual time. */
  async waitForHidden(locator: Locator, timeoutTicks = 600): Promise<void> {
    for (let i = 0; i < timeoutTicks; i++) {
      if (!(await locator.isVisible().catch(() => false))) return;
      await this.tick();
    }
    const shot = path.join(this.opts.outDir, "waitfor-timeout.png");
    await this.page.screenshot({ path: shot }).catch(() => {});
    throw new Error(
      `timed out waiting for ${String(locator)} to hide; state in ${shot}`,
    );
  }

  /** finish writes the timeline and closes the browser, returning the timeline. */
  async finish(): Promise<Timeline> {
    this.endZoom();
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
