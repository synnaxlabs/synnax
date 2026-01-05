// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import {
  type channel,
  control,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import { color, type destructor, observe, unique } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { theming } from "@/theming/aether";

export const stateProviderStateZ = z.object({});

interface InternalState {
  palette: color.Color[];
  instrumentation: Instrumentation;
  defaultColor: color.Color;
  client: Synnax | null;
}

const CONTEXT_KEY = "control-state-provider";

/**
 * An extension of the Synnax client's control state that allows us to assign a unique
 * color to each control subject for user identification.
 */
export const sugaredStateZ = control.stateZ.extend({
  subjectColor: color.colorZ,
});

/**
 * An extension of the Synnax client's control state that allows us to assign a unique
 * color to each control subject for user identification.
 */
export interface SugaredState extends control.State {
  subjectColor: color.Color;
}

/**
 * StateProvider tracks the control state for the channels in a Synnax cluster, listening
 * for updates and providing a way to get the current state for a channel.
 */
export class StateProvider extends aether.Composite<
  typeof stateProviderStateZ,
  InternalState
> {
  static readonly TYPE = "StateProvider";
  schema = stateProviderStateZ;

  /** Tracks the colors we assign to a particular control subject. */
  private readonly colors = new Map<string, color.Color>();

  /** Tracks the current control state for each channel and allows us to access it */
  private tracker?: control.StateTracker;
  /** Stop listening for changes to the tracker */
  private disconnectTrackerChange?: destructor.Destructor;

  /** An observer that lets external subscribers know when the control state changes */
  private readonly obs: observe.Observer<control.Transfer[]> = new observe.Observer();

  /**
   * Grabs the state provider from the current aether context.
   *
   * @param ctx - The component's current aether context.
   * @throws {Error} if the state provider is not in the context.
   */
  static use(ctx: aether.Context): StateProvider {
    return ctx.get(CONTEXT_KEY);
  }

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.instrumentation = alamos.useInstrumentation(ctx, "control-state");
    const theme = theming.use(ctx);
    i.palette = theme.colors.visualization.palettes.default;
    i.defaultColor = theme.colors.gray.l8;
    const nextClient = synnax.use(ctx);
    if (i.client != null && nextClient === i.client) return;
    i.client = nextClient;
    ctx.set(CONTEXT_KEY, this);
    const runAsync = status.useErrorHandler(ctx);
    runAsync(async () => {
      await this.maybeCloseTracker();
      if (i.client == null) return;
      this.internal.instrumentation.L.debug("starting state tracker");
      await this.openTracker(i.client);
    }, "failed to open control state tracker");
  }

  private async maybeCloseTracker(): Promise<void> {
    if (this.tracker == null) return;
    this.internal.instrumentation.L.debug("stopping state tracker");
    this.disconnectTrackerChange?.();
    await this.tracker.close();
    this.tracker = undefined;
  }

  afterDelete(ctx: aether.Context): void {
    const runAsync = status.useErrorHandler(ctx);
    runAsync(async () => {
      await this.maybeCloseTracker();
    }, "failed to close control state tracker");
  }

  onChange(cb: (transfers: control.Transfer[]) => void): destructor.Destructor {
    return this.obs.onChange(cb);
  }

  get(key: channel.Key): SugaredState | undefined;

  get(keys: channel.Key[]): SugaredState[];

  get(key: channel.Key | channel.Key[]): SugaredState | SugaredState[] | undefined {
    if (Array.isArray(key))
      return unique
        .unique(key)
        .map((k) => this.getOne(k))
        .filter((s) => s != null);
    return this.getOne(key);
  }

  private getOne(key: channel.Key): SugaredState | undefined {
    if (this.tracker == null) return undefined;
    const s = this.tracker.states.get(key);
    if (s == null) return undefined;
    return {
      ...s,
      subjectColor: this.colors.get(s.subject.key) ?? this.internal.defaultColor,
    };
  }

  private async openTracker(client: Synnax): Promise<void> {
    const { internal: i } = this;
    const { instrumentation: ins } = i;
    try {
      this.tracker = await client.control.openStateTracker();
    } catch {
      ins.L.error("failed to open state tracker");
      return;
    }
    this.disconnectTrackerChange?.();
    this.disconnectTrackerChange = this.tracker.onChange((t) => {
      ins.L.debug("transfer", { transfers: t.map((t) => control.transferString(t)) });
      if (this.tracker == null)
        throw new UnexpectedError("tracker is null inside it's own onChange callback!");
      this.updateColors(this.tracker);
      this.obs.notify(t);
    });
  }

  private updateColors(t: control.StateTracker): void {
    const sub = t.subjects();
    const subKeys = sub.map((s) => s.key);
    const colors = Array.from(this.colors.values());

    // Purge colors that are no longer in use
    this.colors.forEach((_, key) => !subKeys.includes(key) && this.colors.delete(key));

    // Add colors for new subjects
    const { palette } = this.internal;
    sub.forEach((s) => {
      if (this.colors.has(s.key)) return;
      const color = palette.find((c) => !colors.includes(c)) ?? palette[0];
      this.colors.set(s.key, color);
      colors.push(color);
    });
  }
}
