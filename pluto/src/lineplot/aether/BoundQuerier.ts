// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds } from "@synnaxlabs/x";
import z from "zod";

import { aether } from "@/aether/aether";

export type AxesBounds = Record<string, bounds.Bounds>;

export const boundQuerierStateZ = z.object({});

export interface BoundQuerierState extends z.infer<typeof boundQuerierStateZ> {}

const axesBoundsZ = z.record(z.string(), z.object({ lower: z.number(), upper: z.number() }));

export const boundQuerierMethodsZ = {
  getBounds: z.function({ input: z.tuple([]), output: axesBoundsZ }),
};

export interface BoundQuerierRenderProps {
  getBounds: () => AxesBounds;
}

export class BoundQuerier
  extends aether.Leaf<
    typeof boundQuerierStateZ,
    BoundQuerierRenderProps,
    typeof boundQuerierMethodsZ
  >
  implements aether.HandlersFromSchema<typeof boundQuerierMethodsZ>
{
  static readonly TYPE: string = "Bounds";
  static readonly METHODS = boundQuerierMethodsZ;

  schema = boundQuerierStateZ;
  methods = boundQuerierMethodsZ;

  afterUpdate(_: aether.Context): void {}

  getBounds(): AxesBounds {
    return this.internal.getBounds();
  }

  render(props: BoundQuerierRenderProps): void {
    this.internal.getBounds = props.getBounds;
  }
}
