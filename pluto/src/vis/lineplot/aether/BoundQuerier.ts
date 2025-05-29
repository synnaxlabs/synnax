// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";
import z from "zod";

import { aether } from "@/aether/aether";

const boundsResponseZ = z.object({
  request: z.string(),
  bounds: z.record(z.string(), bounds.bounds),
});

export type AxesBounds = Record<string, bounds.Bounds>;

export const boundQuerierStateZ = z.object({
  request: z.string(),
  response: boundsResponseZ,
});

export interface BoundQuerierState extends z.infer<typeof boundQuerierStateZ> {}

export interface BoundQuerierRenderProps {
  getBounds: () => AxesBounds;
}

export class BoundQuerier extends aether.Leaf<
  typeof boundQuerierStateZ,
  BoundQuerierRenderProps
> {
  static readonly TYPE: string = "Bounds";
  schema = boundQuerierStateZ;

  afterUpdate(_: aether.Context): void {
    if (this.state.request !== this.prevState.request)
      this.setState((p) => ({
        ...p,
        response: {
          request: p.request,
          bounds: this.internal.getBounds(),
        },
      }));
  }

  render(props: BoundQuerierRenderProps): void {
    this.internal.getBounds = props.getBounds;
  }
}
