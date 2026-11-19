// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control, type Synnax } from "@synnaxlabs/client";
import { color, type destructor, observe } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color as aetherColor } from "@/color/aether";
import { synnax } from "@/synnax/aether";
import { theming } from "@/theming/aether";

export const colorsStateZ = z.object({});

interface InternalState {
  palette: color.Color[];
  defaultColor: color.Color;
  client: Synnax | null;
}

const CONTEXT_KEY = "control-colors";

/**
 * A control state with the color assigned to the subject holding it, which is how
 * control state reaches the main thread.
 */
export const coloredStateZ = control.stateZ.extend({ subjectColor: color.colorZ });

export interface ColoredState extends control.State {
  subjectColor: color.Color;
}

/**
 * Colors assigns each control subject a series color, so that a reader can tell
 * subjects apart at a glance. A subject takes the first series color no other subject
 * holds, and keeps it while it holds control. Two subjects share a color only when more
 * subjects hold control than there are series colors. When a color frees up, or the
 * series colors change, the later of two subjects that share a color moves to a free
 * one.
 */
export class Colors extends aether.Composite<typeof colorsStateZ, InternalState> {
  static readonly TYPE = "Colors";
  static readonly stateZ = colorsStateZ;
  schema = Colors.stateZ;

  /**
   * The number assigned to each control subject, keyed by subject key. A subject takes
   * the series color at its number, wrapping around.
   */
  private readonly assigned = new Map<string, number>();
  /** User-specified color overrides from the legend UI. */
  private readonly overrides = new Map<string, color.Color>();
  private readonly obs = new observe.Observer<void>();
  private disconnect?: destructor.Destructor;

  /**
   * Grabs the color assignment from the current aether context.
   *
   * @param ctx - The component's current aether context.
   * @throws {Error} if it is not in the context.
   */
  static use(ctx: aether.Context): Colors {
    return ctx.get(CONTEXT_KEY);
  }

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.palette = aetherColor.series.use(ctx);
    i.defaultColor = theming.use(ctx).colors.gray.l9;
    ctx.set(CONTEXT_KEY, this);
    const nextClient = synnax.use(ctx);
    if (i.client != null && nextClient === i.client) {
      this.settle(new Set(this.assigned.keys()));
      return this.obs.notify();
    }
    i.client = nextClient;
    this.disconnect?.();
    this.assigned.clear();
    if (nextClient == null) return;
    const { store } = nextClient.control;
    const assign = () => this.assign(store.get());
    this.disconnect = store.subscribeBatch(assign);
    assign();
  }

  afterDelete(): void {
    this.disconnect?.();
  }

  /** Subscribes to changes in the assignment. The handler takes no arguments: it
   * re-reads the colors it needs through {@link get}. */
  onChange(handler: () => void): destructor.Destructor {
    return this.obs.onChange(handler);
  }

  /** The color of the given subject, or the default color when it has none. */
  get(subject: string): color.Color {
    const { defaultColor, palette } = this.internal;
    const override = this.overrides.get(subject);
    if (override != null) return override;
    const n = this.assigned.get(subject);
    if (n == null || palette.length === 0) return defaultColor;
    return palette[n % palette.length];
  }

  setOverrides(overrides: Record<string, color.Color>): void {
    const entries = Object.entries(overrides);
    if (
      entries.length === this.overrides.size &&
      entries.every(([k, v]) => {
        const existing = this.overrides.get(k);
        return existing != null && color.equals(existing, v);
      })
    )
      return;
    this.overrides.clear();
    for (const [key, value] of entries) this.overrides.set(key, color.construct(value));
    this.obs.notify();
  }

  /**
   * Assigns a color to every subject holding control, and releases the colors of
   * subjects that hold none. Colors are assigned over the subjects the client has
   * cached, so an unrelated subject elsewhere in the cluster never shifts them.
   */
  private assign(states: control.KeyedState[]): void {
    this.settle(new Set(states.map((s) => s.subject.key)));
    this.obs.notify();
  }

  /**
   * Gives each of the subjects a number. A held number stays unless an earlier number
   * holds its color and a color is free. A new subject takes the first free color, or
   * the smallest free number when no color is free.
   */
  private settle(subjects: Set<string>): void {
    const { length } = this.internal.palette;
    const colors = new Set<number>();
    const numbers = new Set<number>();
    const take = (key: string, n: number): void => {
      this.assigned.set(key, n);
      numbers.add(n);
      if (length > 0) colors.add(n % length);
    };
    const freeColor = (): number | undefined => {
      for (let n = 0; n < length; n++) if (!colors.has(n)) return n;
      return undefined;
    };
    const freeNumber = (): number => {
      let n = 0;
      while (numbers.has(n)) n++;
      return n;
    };
    const held = [...this.assigned]
      .filter(([key]) => subjects.has(key))
      .sort(([, a], [, b]) => a - b);
    this.assigned.clear();
    const shared: Array<[string, number]> = [];
    for (const [key, n] of held)
      if (length > 0 && colors.has(n % length)) shared.push([key, n]);
      else take(key, n);
    shared.forEach(([key, n]) => take(key, freeColor() ?? n));
    subjects.forEach((key) => {
      if (!this.assigned.has(key)) take(key, freeColor() ?? freeNumber());
    });
  }
}
