// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { type Synnax, control, type channel } from "@synnaxlabs/client";
import { observe, type Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { type color } from "@/color/core";
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

export class StateProvider extends aether.Composite<
  typeof stateProviderStateZ,
  InternalState
> {
  static readonly TYPE = "StateProvider";
  schema = stateProviderStateZ;

  private readonly defaultState = new Map<channel.Key, control.State>();
  private readonly colors = new Map<string, color.Color>();

  tracker?: control.StateTracker;
  private disconnectTrackerChange?: Destructor;

  private readonly obs: observe.Observer<control.Transfer[]> = new observe.Observer();

  static use(ctx: aether.Context): StateProvider {
    return ctx.get(CONTEXT_KEY);
  }

  afterUpdate(): void {
    this.internal.instrumentation = alamos.useInstrumentation(
      this.ctx,
      "control-state",
    );
    const theme = theming.use(this.ctx);
    this.internal.palette = theme.colors.visualization.palettes.default;
    this.internal.defaultColor = theme.colors.gray.l6;
    const nextClient = synnax.use(this.ctx);
    if (nextClient === this.internal.client) return;
    this.internal.client = nextClient;
    this.ctx.set(CONTEXT_KEY, this);
    if (this.internal.client != null) {
      this.internal.instrumentation.L.debug("starting state tracker");
      void this.startUpdating(this.internal.client);
    }
  }

  afterDelete(): void {
    this.disconnectTrackerChange?.();
    this.tracker?.close().catch(this.internal.instrumentation.L.error);
  }

  onChange(cb: (transfers: control.Transfer[]) => void): Destructor {
    return this.obs.onChange(cb);
  }

  get controlState(): Map<channel.Key, control.State> {
    return this.tracker?.states ?? this.defaultState;
  }

  getColor(channel: channel.Key): color.Color {
    const df = this.internal.defaultColor;
    const state = this.controlState.get(channel);
    return state == null ? df : this.colors.get(state.subject.key) ?? df;
  }

  private async startUpdating(client: Synnax): Promise<void> {
    const { instrumentation: i } = this.internal;
    try {
      this.tracker = await control.StateTracker.open(client.telem);
    } catch {
      i.L.error("failed to open state tracker");
      return;
    }
    this.disconnectTrackerChange = this.tracker.onChange((t) => {
      i.L.debug("transfer", { transfers: t.map((t) => control.transferString(t)) });
      this.updateColors(this.tracker as control.StateTracker);
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
