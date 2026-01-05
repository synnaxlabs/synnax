// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { z } from "zod";

import { aether } from "@/aether/aether";

export const providerStateZ = z.object({
  include: z.string().array().optional(),
  exclude: z.string().array().optional(),
  level: z.enum(alamos.LOG_LEVELS).default("info"),
});
export type ProviderState = z.input<typeof providerStateZ>;

const CONTEXT_KEY = "alamos-provider";

export interface InternalState {
  ins: alamos.Instrumentation;
}

export class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
  static readonly TYPE = "alamos.Provider";
  schema = providerStateZ;

  afterUpdate(ctx: aether.Context): void {
    const v = ctx.getOptional<alamos.Instrumentation>(CONTEXT_KEY);
    if (v != null) return;

    const filters: alamos.LogLevelFilter[] = [];
    const { include, exclude, level } = this.state;
    if (include != null || exclude != null)
      filters.push(alamos.logLevelKeyFiler({ include, exclude }));
    if (level != null) filters.push(alamos.logThresholdFilter(level));
    if (filters.length === 0) filters.push(() => false);

    this.internal.ins = new alamos.Instrumentation({
      key: "aether",
      logger: new alamos.Logger({ filters }),
    });
    ctx.set(CONTEXT_KEY, this.internal.ins);
  }
}

export const useInstrumentation = (
  ctx: aether.Context,
  name?: string,
): alamos.Instrumentation => {
  const ins = ctx.get<alamos.Instrumentation>(CONTEXT_KEY);
  if (ins == null) throw new Error("No instrumentation provider");
  return name == null ? ins : ins.child(name);
};

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};
