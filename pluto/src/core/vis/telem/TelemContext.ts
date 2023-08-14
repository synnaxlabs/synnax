// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Destructor } from "@synnaxlabs/x";

import { TelemSpec } from "./telem";

import { AetherContext } from "@/core/aether/worker";

export interface TelemProvider {
  use: <T>(key: string, props: TelemSpec) => UseTelemResult<T>;
}

export type UseTelemResult<T> = [T, Destructor];

export class TelemContext {
  private static readonly CONTEXT_KEY = "pluto-telem-context";

  prov: TelemProvider;

  private constructor(prov: TelemProvider) {
    this.prov = prov;
  }

  static set(ctx: AetherContext, prov: TelemProvider): void {
    const telem = new TelemContext(prov);
    ctx.set(TelemContext.CONTEXT_KEY, telem);
  }

  static use<T>(ctx: AetherContext, key: string, props: TelemSpec): UseTelemResult<T> {
    return ctx.get<TelemContext>(TelemContext.CONTEXT_KEY).prov.use<T>(key, props);
  }
}
