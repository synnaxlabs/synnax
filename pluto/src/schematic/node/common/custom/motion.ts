// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { deep, type dimensions, type location, TimeSpan, xy } from "@synnaxlabs/x";

import type { Loop, Mover } from "@/schematic/node/common/custom/loop";
import { matched } from "@/schematic/node/common/custom/paint";

/** Reading is what one state of a symbol reads. */
export interface Reading {
  /** value is the value of the state, or NaN while it is unknown. */
  value: number;
  /** stale is true while the data of the state is stale. */
  stale?: boolean;
}

export interface MotionParams {
  loop: Loop;
  /** reduced is true under the operating system's Reduce motion setting. */
  reduced: boolean;
  /**
   * readings holds the reading of each state by key. A state with no reading reads as
   * a state with no source: its default entry, or fraction 0.
   */
  readings: Record<string, Reading>;
  disabled: boolean;
}

/** @returns whether a look moves its part. */
export const moves = ({ motion, blink }: schematic.symbol.Look): boolean =>
  motion != null || blink != null;

// One flash is one period, so a blink stays under three flashes a second.
const MIN_BLINK_PERIOD = TimeSpan.milliseconds(400).milliseconds;
const BLINK_DIM = 0.2;
// The duration and curve of --pluto-modal-animation.
const TRANSITION_MS = 150;

// easeInOut is the CSS ease-in-out curve, cubic-bezier(X1, 0, X2, 1), at time t.
const X1 = 0.42;
const X2 = 0.58;
const easeInOut = (t: number): number => {
  let lo = 0;
  let hi = 1;
  for (let i = 0; i < 20; i++) {
    const s = (lo + hi) / 2;
    if (3 * X1 * s * (1 - s) ** 2 + 3 * X2 * s * s * (1 - s) + s ** 3 < t) lo = s;
    else hi = s;
  }
  const s = (lo + hi) / 2;
  return s * s * (3 - 2 * s);
};

type Kind = schematic.symbol.MotionType | "blink";
type Of<K extends Kind> = K extends "blink"
  ? schematic.symbol.Blink
  : Extract<schematic.symbol.Motion, { type: K }>;

const dashCycle = (el: Element): number => {
  const lengths = getComputedStyle(el)
    .strokeDasharray.split(/[\s,]+/)
    .map(Number.parseFloat)
    .filter(Number.isFinite);
  const sum = lengths.reduce((a, b) => a + b, 0);
  return lengths.length % 2 === 1 ? 2 * sum : sum;
};

const num = (v: number): string => String(Math.round(v * 1000) / 1000);

// Pose is what one motion that writes transform gives an element.
interface Pose {
  /** offset is a move, as fractions of the box. */
  offset?: xy.XY;
  /** angle is a turn about the pivot of the part, in degrees. */
  angle?: number;
}

/** Shape is one element of a moving part, for one attribute that its motions write. */
class Shape {
  /** written is the value that the motions last wrote. */
  written?: string;
  private cycle?: number;
  private turn?: { of: xy.XY; at: xy.XY };
  private readonly poses = new Map<Kind, Pose>();

  constructor(
    readonly el: SVGElement,
    /** own is the value of the attribute before the motion wrote it. */
    readonly own: string | null,
    /** matrix maps fractions of the box to the coordinates of the parent of el. */
    readonly matrix: DOMMatrix,
    /** pivotOf returns the pivot of the part as fractions of the box. */
    private readonly pivotOf: () => xy.XY,
  ) {}

  /** dash is the length of one cycle of the dash pattern of the element. */
  get dash(): number {
    return (this.cycle ??= dashCycle(this.el));
  }

  /** pivot is the pivot of the part in the coordinates of the parent of el. */
  get pivot(): xy.XY {
    const of = this.pivotOf();
    if (this.turn == null || this.turn.of !== of) {
      const { x, y } = this.matrix.transformPoint(of);
      this.turn = { of, at: { x, y } };
    }
    return this.turn.at;
  }

  /** @returns the transform of every pose, after the motion of kind takes pose. */
  pose(kind: Kind, pose: Pose): string {
    this.poses.set(kind, pose);
    return this.transform();
  }

  /** @returns the value to draw after the motion of kind lets go of the element. */
  release(kind: Kind): string | null {
    this.poses.delete(kind);
    return this.poses.size === 0 ? this.own : this.transform();
  }

  // transform moves the element, then turns it about the pivot, then applies own.
  private transform(): string {
    let offset: xy.XY | undefined;
    let angle: number | undefined;
    for (const pose of this.poses.values()) {
      if (pose.offset != null) offset = xy.translate(offset ?? xy.ZERO, pose.offset);
      if (pose.angle != null) angle = (angle ?? 0) + pose.angle;
    }
    const list: string[] = [];
    if (offset != null) {
      const { a, b, c, d } = this.matrix;
      const { x, y } = offset;
      list.push(`translate(${num(a * x + c * y)} ${num(b * x + d * y)})`);
    }
    if (angle != null) {
      const { x, y } = this.pivot;
      list.push(`rotate(${num(angle)} ${num(x)} ${num(y)})`);
    }
    if (this.own != null) list.push(this.own);
    return list.join(" ");
  }
}

const parse = (own: string | null, fallback: number): number => {
  const v = Number.parseFloat(own ?? "");
  return Number.isFinite(v) ? v : fallback;
};
const lerp = (from: number, to: number, f: number): number => from + (to - from) * f;

const INSETS: Record<location.Outer, (cut: string) => string> = {
  bottom: (cut) => `inset(${cut} 0 0 0)`,
  top: (cut) => `inset(0 0 ${cut} 0)`,
  left: (cut) => `inset(0 ${cut} 0 0)`,
  right: (cut) => `inset(0 0 0 ${cut})`,
};

type Attribute = "transform" | "stroke-dashoffset" | "clip-path" | "opacity";

interface Entry<M> {
  /** attribute is the attribute that the motion writes. No look writes it. */
  attribute: Attribute;
  /**
   * speed is the cycles each second at the upper bound of the state. A motion with no
   * speed follows the fraction of its state.
   */
  speed?: (motion: M) => number;
  /** held keeps the last cycle drawn while the motion stops. Otherwise it draws 0. */
  held?: boolean;
  /** draw returns the attribute value at a cycle in [0, 1), or at a fraction. */
  draw: (motion: M, shape: Shape, value: number) => string;
}

const MOTIONS: { [K in Kind]: Entry<Of<K>> } = {
  spin: {
    attribute: "transform",
    speed: ({ rate }) => rate,
    held: true,
    draw: (_, shape, turns) => shape.pose("spin", { angle: turns * 360 }),
  },
  flow: {
    attribute: "stroke-dashoffset",
    speed: ({ rate }) => rate,
    held: true,
    draw: (_, shape, cycles) => num(parse(shape.own, 0) - cycles * shape.dash),
  },
  blink: {
    attribute: "opacity",
    speed: ({ period }) =>
      1000 / Math.max(new TimeSpan(period).milliseconds, MIN_BLINK_PERIOD),
    draw: (_, shape, cycles) =>
      num(parse(shape.own, 1) * (cycles < 0.5 ? 1 : BLINK_DIM)),
  },
  level: {
    attribute: "clip-path",
    draw: ({ side }, _, f) => INSETS[side](`${num((1 - f) * 100)}%`),
  },
  rotate: {
    attribute: "transform",
    draw: ({ from, to }, shape, f) =>
      shape.pose("rotate", { angle: lerp(from, to, f) }),
  },
  move: {
    attribute: "transform",
    draw: ({ from, to }, shape, f) =>
      shape.pose("move", {
        offset: { x: lerp(from.x, to.x, f), y: lerp(from.y, to.y, f) },
      }),
  },
};

const transformOf = (el: Element): DOMMatrix => {
  const m = new DOMMatrix();
  if (!(el instanceof SVGGraphicsElement)) return m;
  const list = el.transform.baseVal;
  for (let i = 0; i < list.numberOfItems; i++) m.multiplySelf(list.getItem(i).matrix);
  return m;
};

// Space is a drawing, its box, and the transforms of its elements before any motion.
interface Space {
  drawing: SVGGElement;
  box: dimensions.Dimensions;
  rest: Map<Element, DOMMatrix>;
}

// toDrawing maps the coordinates of the parent of el to the coordinates of drawing.
const toDrawing = (el: Element, { drawing, rest }: Space): DOMMatrix => {
  const m = new DOMMatrix();
  if (el === drawing) return m;
  for (let a: Element | null = el.parentElement; a != null && a !== drawing;) {
    const t = rest.get(a);
    if (t != null) m.preMultiplySelf(t);
    a = a.parentElement;
  }
  return m;
};

const MIDDLE: xy.XY = { x: 0.5, y: 0.5 };

// center returns the center of the bounding box of elements, as fractions of the box,
// or undefined while no element has a layout, as in a hidden tab.
const center = (elements: SVGElement[], space: Space): xy.XY | undefined => {
  const graphics = elements.filter((el) => el instanceof SVGGraphicsElement);
  if (graphics.length === 0) return MIDDLE;
  const xs: number[] = [];
  const ys: number[] = [];
  for (const el of graphics) {
    const { x, y, width, height } = el.getBBox();
    if (width === 0 && height === 0) continue;
    const m = toDrawing(el, space);
    const own = space.rest.get(el);
    if (own != null) m.multiplySelf(own);
    for (const corner of [
      { x, y },
      { x: x + width, y },
      { x, y: y + height },
      { x: x + width, y: y + height },
    ]) {
      const p = m.transformPoint(corner);
      xs.push(p.x);
      ys.push(p.y);
    }
  }
  if (xs.length === 0) return undefined;
  return {
    x: (Math.min(...xs) + Math.max(...xs)) / 2 / space.box.width,
    y: (Math.min(...ys) + Math.max(...ys)) / 2 / space.box.height,
  };
};

interface Context {
  states: Map<string, schematic.symbol.State>;
  readings: Record<string, Reading>;
  frozen: boolean;
}

// stops is true while a reading is stale or unknown.
const stops = ({ readings }: Context, key: string): boolean => {
  const reading = readings[key];
  return reading != null && (reading.stale === true || Number.isNaN(reading.value));
};

const fraction = ({ states, readings }: Context, key: string): number => {
  const state = states.get(key);
  const value = readings[key]?.value ?? NaN;
  if (state?.type !== "number" || Number.isNaN(value)) return 0;
  const { lower, upper } = state.bounds;
  // The Core refuses these bounds, but a draft in the editor can hold them.
  if (upper <= lower) return 0;
  return Math.min(1, Math.max(0, (value - lower) / (upper - lower)));
};

interface Effect {
  value: schematic.symbol.Motion | schematic.symbol.Blink;
  /** reads are the keys of the states that the look reads in `when` or `state`. */
  reads: string[];
}

interface Effects {
  /** motions holds the motion of each kind that applies, blink included. */
  motions: Map<Kind, Effect>;
  hidden: boolean;
}

const applies = (look: schematic.symbol.Look, ctx: Context): boolean => {
  if (look.when == null) return true;
  const state = ctx.states.get(look.when.state);
  if (state == null) return false;
  const value = ctx.readings[look.when.state]?.value ?? NaN;
  const read =
    state.type === "enumeration" && Number.isNaN(value)
      ? state.default
      : matched(state, value);
  return look.when.values.includes(read);
};

// resolve returns the motions that the looks in effect give each part, by part key. A
// later look replaces an earlier motion of its kind only. The key undefined is the
// whole drawing.
const resolve = (
  looks: schematic.symbol.Look[],
  ctx: Context,
): Map<string | undefined, Effects> => {
  const applied = looks.filter((look) => applies(look, ctx));
  const effects = new Map<string | undefined, Effects>();
  for (const look of applied) {
    if (look.motion == null && look.blink == null) continue;
    const effect = effects.get(look.part) ?? {
      motions: new Map<Kind, Effect>(),
      hidden: false,
    };
    const reads = look.when == null ? [] : [look.when.state];
    if (look.motion != null)
      effect.motions.set(look.motion.type, { value: look.motion, reads });
    if (look.blink != null) effect.motions.set("blink", { value: look.blink, reads });
    effects.set(look.part, effect);
  }
  for (const [part, effect] of effects)
    effect.hidden =
      applied.findLast((l) => l.hidden != null && (l.part == null || l.part === part))
        ?.hidden === true;
  return effects;
};

type Value = Effect["value"];

// Track moves the elements of a part with one motion.
class Track {
  private readonly entry: Entry<Value>;
  // value is the cycle of a motion that cycles, or the fraction drawn.
  private value = 0;
  private speed = 0;
  private last?: number;
  private from = 0;
  private target = 0;
  private start?: number;
  // startEase is how far along its curve the slide was when it took its target.
  private startEase = 0;

  constructor(
    private readonly kind: Kind,
    private motion: Value,
    private readonly shapes: Shape[],
  ) {
    this.entry = MOTIONS[kind] as Entry<Value>;
  }

  get moving(): boolean {
    return this.entry.speed != null ? this.speed !== 0 : this.value !== this.target;
  }

  set({ value, reads }: Effect, ctx: Context, fresh: boolean): void {
    this.motion = value;
    const state = "state" in value ? value.state : undefined;
    const { speed, held } = this.entry;
    if (speed == null) {
      if (state != null) this.follow(ctx, state, fresh);
      return;
    }
    const stopped =
      ctx.frozen ||
      reads.some((key) => stops(ctx, key)) ||
      (state != null && stops(ctx, state));
    const next = stopped
      ? 0
      : speed(value) * (state == null ? 1 : fraction(ctx, state));
    if (next === 0) {
      this.last = undefined;
      if (this.speed !== 0 && held !== true) {
        this.value = 0;
        this.draw();
      }
    }
    this.speed = next;
  }

  // follow jumps to the fraction of state when fresh, frozen, or unknown, and keeps its
  // target while state is stale. Otherwise it starts a transition to the fraction. A
  // part that does not move draws again, for a look whose fields changed.
  private follow(ctx: Context, state: string, fresh: boolean): void {
    const reading = ctx.readings[state];
    const held = !fresh && reading?.stale === true;
    const f = held ? this.target : fraction(ctx, state);
    if (!held && (fresh || ctx.frozen || Number.isNaN(reading?.value ?? 0))) {
      this.value = f;
      this.target = f;
      this.draw();
    } else if (f !== this.target) {
      // A running slide keeps its clock, so a value that streams in faster than the
      // slide still lands within TRANSITION_MS of its arrival.
      const { start, last } = this;
      const running = this.moving && start != null && last != null;
      this.startEase = running ? easeInOut((last - start) / TRANSITION_MS) : 0;
      if (!running) this.start = undefined;
      this.from = this.value;
      this.target = f;
    } else if (!this.moving) this.draw();
  }

  step(now: number): boolean {
    if (!this.moving) return false;
    this.value = this.entry.speed == null ? this.ease(now) : this.cycle(now);
    this.draw();
    return this.moving;
  }

  private ease(now: number): number {
    this.start ??= now;
    this.last = now;
    const t = Math.min(1, (now - this.start) / TRANSITION_MS);
    if (t === 1) return this.target;
    const left = (1 - easeInOut(t)) / (1 - this.startEase);
    return lerp(this.target, this.from, left);
  }

  private cycle(now: number): number {
    const dt = this.last == null ? 0 : (now - this.last) / 1000;
    this.last = now;
    return (((this.value + this.speed * dt) % 1) + 1) % 1;
  }

  private draw(): void {
    const { attribute, draw } = this.entry;
    for (const shape of this.shapes) {
      const v = draw(this.motion, shape, this.value);
      if (v === shape.written) continue;
      shape.el.setAttribute(attribute, v);
      shape.written = v;
    }
  }

  restore(): void {
    const { attribute } = this.entry;
    for (const shape of this.shapes) {
      const v = shape.release(this.kind);
      if (v == null) shape.el.removeAttribute(attribute);
      else shape.el.setAttribute(attribute, v);
      shape.written = v ?? undefined;
    }
  }
}

// Part moves the elements of one part, or of the whole drawing, with one track for each
// kind of motion.
class Part implements Mover {
  private readonly tracks = new Map<Kind, Track>();
  private readonly shapes = new Map<Attribute, Shape[]>();
  private center?: xy.XY;

  constructor(
    readonly def: schematic.symbol.Part | undefined,
    private readonly elements: SVGElement[],
    private readonly space: Space,
  ) {}

  get moving(): boolean {
    for (const track of this.tracks.values()) if (track.moving) return true;
    return false;
  }

  // A part with no pivot turns about the center of its own box. A part with no layout
  // yet measures again on the next frame.
  private readonly pivotOf = (): xy.XY =>
    this.def?.pivot ?? (this.center ??= center(this.elements, this.space)) ?? MIDDLE;

  // The motions that write one attribute share its shapes, so each element keeps the
  // attribute as drawn before any of them.
  private shapesOf(attribute: Attribute): Shape[] {
    let shapes = this.shapes.get(attribute);
    if (shapes != null) return shapes;
    const { width, height } = this.space.box;
    shapes = this.elements.map((el) => {
      const matrix = toDrawing(el, this.space).invertSelf().scaleSelf(width, height);
      return new Shape(el, el.getAttribute(attribute), matrix, this.pivotOf);
    });
    this.shapes.set(attribute, shapes);
    return shapes;
  }

  set(effects: Effects, ctx: Context): void {
    const partCtx = { ...ctx, frozen: ctx.frozen || effects.hidden };
    for (const [kind, track] of this.tracks) {
      if (effects.motions.has(kind)) continue;
      track.restore();
      this.tracks.delete(kind);
    }
    for (const [kind, effect] of effects.motions) {
      let track = this.tracks.get(kind);
      const fresh = track == null;
      track ??= new Track(kind, effect.value, this.shapesOf(MOTIONS[kind].attribute));
      track.set(effect, partCtx, fresh);
      this.tracks.set(kind, track);
    }
  }

  step(now: number): boolean {
    let moving = false;
    // Every track steps, so no short circuit.
    for (const track of this.tracks.values()) if (track.step(now)) moving = true;
    return moving;
  }

  restore(): void {
    for (const track of this.tracks.values()) track.restore();
  }
}

const select = (drawing: SVGGElement, def: schematic.symbol.Part): SVGElement[] => [
  ...new Set(
    def.selectors.flatMap((s) => Array.from(drawing.querySelectorAll<SVGElement>(s))),
  ),
];

/**
 * Motion moves the parts of one drawing on a loop, by the looks of its spec. Copy the
 * spec on change and never change it in place.
 */
export class Motion {
  private readonly parts = new Map<string | undefined, Part>();
  private readonly space: Space;

  /** Builds a Motion for a drawing that nothing has moved yet. */
  constructor(
    readonly loop: Loop,
    drawing: SVGGElement,
    box: dimensions.Dimensions,
  ) {
    const rest = new Map<Element, DOMMatrix>();
    for (const el of drawing.querySelectorAll("[transform]"))
      rest.set(el, transformOf(el));
    this.space = { drawing, box, rest };
  }

  update(
    spec: schematic.symbol.Spec,
    { readings, reduced, disabled }: MotionParams,
  ): void {
    const ctx: Context = {
      states: new Map(spec.states.map((s) => [s.key, s])),
      readings,
      frozen: reduced || disabled,
    };
    const effects = resolve(spec.looks, ctx);
    const defs = new Map(spec.parts.map((p) => [p.key, p]));
    const defOf = (key: string | undefined) =>
      key == null ? undefined : defs.get(key);
    for (const [key, part] of this.parts) {
      if (effects.has(key) && deep.equal(part.def, defOf(key))) continue;
      this.loop.delete(part);
      part.restore();
      this.parts.delete(key);
    }
    for (const [key, effect] of effects) {
      let part = this.parts.get(key);
      if (part == null) {
        const def = defOf(key);
        const { drawing } = this.space;
        const elements =
          key == null ? [drawing] : def == null ? [] : select(drawing, def);
        part = new Part(def, elements, this.space);
        this.parts.set(key, part);
      }
      part.set(effect, ctx);
      if (part.moving) this.loop.add(part);
      else this.loop.delete(part);
    }
  }

  /** Takes every part out of the loop and draws the drawing as drawn. */
  stop(): void {
    for (const part of this.parts.values()) {
      this.loop.delete(part);
      part.restore();
    }
    this.parts.clear();
  }
}
