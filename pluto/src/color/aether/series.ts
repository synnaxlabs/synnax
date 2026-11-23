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

/** SeriesProvider gives its descendants the series colors, resolved for the theme. */
export class SeriesProvider extends aether.Composite<typeof providerStateZ> {
  static readonly TYPE = "color.SeriesProvider";
  static readonly z = providerStateZ;
  override schema = SeriesProvider.z;

  override afterUpdate(ctx: aether.Context): void {
    ctx.set(CONTEXT_KEY, this.state.colors);
  }
}

/**
 * @returns the series colors in palette order. The list is empty until they load.
 * @throws {NotFoundError} if no {@link SeriesProvider} is above the component.
 */
export const useSeries = (ctx: aether.Context): color.Color[] =>
  ctx.get<color.Color[]>(CONTEXT_KEY);

export const REGISTRY: aether.ComponentRegistry = {
  [SeriesProvider.TYPE]: SeriesProvider,
};
