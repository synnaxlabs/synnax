// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, control, type channel } from "@synnaxlabs/client";
import { observe, type Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { type color } from "@/color/core";
import { synnax } from "@/synnax/aether";
import { theming } from "@/theming/aether";

export const stateProviderStateZ = z.object({});

interface InternalState {
  theme: theming.Theme;
  client: Synnax | null;
}

const CONTEXT_KEY = "control-state-provider";

export class StateProvider extends aether.Composite<
  typeof stateProviderStateZ,
  InternalState
> {
  static readonly TYPE = "StateProvider";
  private onChangeDestroy?: Destructor;
  schema = stateProviderStateZ;
  private readonly defaultState = new Map<channel.Key, control.State>();
  private readonly colors = new Map<string, color.Color>();
  tracker?: control.StateTracker;
  private readonly obs: observe.Observer<control.Transfer[]> = new observe.Observer();

  static use(ctx: aether.Context): StateProvider {
    return ctx.get(CONTEXT_KEY);
  }

  afterUpdate(): void {
    this.internal.theme = theming.use(this.ctx);
    const nextClient = synnax.use(this.ctx);
    if (nextClient == null || nextClient === this.internal.client) return;
    this.internal.client = nextClient;
    this.ctx.set(CONTEXT_KEY, this);
    void this.execUpdate(this.internal.client);
  }

  afterDelete(): void {
    this.onChangeDestroy?.();
    this.tracker?.close();
  }

  onChange(cb: (transfers: control.Transfer[]) => void): Destructor {
    console.log("REGISTERING CHANGE CALLBACK", cb);
    return this.obs.onChange(cb);
  }

  get controlState(): Map<channel.Key, control.State> {
    return this.tracker?.states ?? this.defaultState;
  }

  getColor(channel: channel.Key): color.Color {
    console.log("GETTING COLOR", channel, this.controlState);
    const state = this.controlState.get(channel);
    if (state == null) return this.internal.theme.colors.gray.p0;
    return this.colors.get(state.subject.key) ?? this.internal.theme.colors.gray.p0;
  }

  private async execUpdate(client: Synnax): Promise<void> {
    this.tracker = await control.StateTracker.open(client.telem);
    this.onChangeDestroy = this.tracker.onChange((t) => {
      this.updateColors(this.tracker as control.StateTracker);
      this.obs.notify(t);
    });
  }

  private updateColors(t: control.StateTracker): void {
    console.log("UPDATING STATE TRACKER COLORS", this.internal.theme);
    const subjects = t.subjects();
    const subjectKeys = subjects.map((s) => s.key);
    const colors = Array.from(this.colors.values());
    const palette = this.internal.theme.colors.visualization.palettes.default;
    this.colors.forEach((_, key) => {
      if (!subjectKeys.includes(key)) this.colors.delete(key);
    });
    subjects.forEach((s) => {
      if (this.colors.has(s.key)) return;
      const color = palette.find((c) => !colors.includes(c)) ?? palette[0];
      this.colors.set(s.key, color);
      colors.push(color);
    });
    console.log(this.colors);
  }
}
