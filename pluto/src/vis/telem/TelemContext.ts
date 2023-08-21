// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Destructor } from "@synnaxlabs/x";

import { aether } from "@/aether/aether";
import { TelemFactory } from "@/telem/factory";

import { TelemSpec } from "./telem";

export interface TelemProvider {
  use: <T>(
    key: string,
    props: TelemSpec,
    extension?: TelemFactory
  ) => UseTelemResult<T>;
}

export type UseTelemResult<T> = [T, Destructor];

export class TelemContext {
  static readonly CONTEXT_KEY = "pluto-telem-context";

  prov: TelemProvider;

  private constructor(prov: TelemProvider) {
    this.prov = prov;
  }

  static get(ctx: aether.Context): TelemContext {
    return ctx.get<TelemContext>(TelemContext.CONTEXT_KEY);
  }

  static set(ctx: aether.Context, prov: TelemProvider): void {
    const telem = new TelemContext(prov);
    ctx.set(TelemContext.CONTEXT_KEY, telem);
  }

  static use<T>(
    ctx: aether.Context,
    key: string,
    props: TelemSpec,
    extension?: TelemFactory
  ): UseTelemResult<T> {
    return TelemContext.get(ctx).prov.use<T>(key, props, extension);
  }
}
