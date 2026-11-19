// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";

const CONTEXT_KEY = "pluto-color-series";

const providerStateZ = z.object({ colors: z.array(color.colorZ) });

/** Provider gives its descendants the series colors, resolved for the theme. */
export class Provider extends aether.Composite<typeof providerStateZ> {
  static readonly TYPE = "color.series.Provider";
  static readonly z = providerStateZ;
  schema = Provider.z;

  afterUpdate(ctx: aether.Context): void {
    ctx.set(CONTEXT_KEY, this.state.colors);
  }
}

/**
 * @returns the series colors in palette order. The list is empty until they load.
 * @throws {NotFoundError} if no {@link Provider} is above the component.
 */
export const use = (ctx: aether.Context): color.Color[] =>
  ctx.get<color.Color[]>(CONTEXT_KEY);

export const REGISTRY: aether.ComponentRegistry = { [Provider.TYPE]: Provider };
