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
 * Colors assigns each control subject a color, so that a reader can tell subjects
 * apart at a glance. Assignment is sequential over the visualization palette, which
 * keeps colors unique for as many subjects as the palette holds.
 */
export class Colors extends aether.Composite<typeof colorsStateZ, InternalState> {
  static readonly TYPE = "Colors";
  static readonly stateZ = colorsStateZ;
  schema = Colors.stateZ;

  /** The color assigned to each control subject, keyed by subject key. */
  private readonly assigned = new Map<string, color.Color>();
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
    const theme = theming.use(ctx);
    i.palette = theme.colors.visualization.palettes.default;
    i.defaultColor = theme.colors.gray.l9;
    ctx.set(CONTEXT_KEY, this);
    const nextClient = synnax.use(ctx);
    if (i.client != null && nextClient === i.client) return;
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
    const { defaultColor } = this.internal;
    return this.overrides.get(subject) ?? this.assigned.get(subject) ?? defaultColor;
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
    const subjects = new Set(states.map((s) => s.subject.key));
    this.assigned.forEach((_, key) => {
      if (!subjects.has(key)) this.assigned.delete(key);
    });
    const taken = Array.from(this.assigned.values());
    const { palette } = this.internal;
    subjects.forEach((key) => {
      if (this.assigned.has(key)) return;
      const next = palette.find((c) => !taken.includes(c)) ?? palette[0];
      this.assigned.set(key, next);
      taken.push(next);
    });
    this.obs.notify();
  }
}
