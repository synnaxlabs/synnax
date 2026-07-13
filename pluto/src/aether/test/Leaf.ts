// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";

export const leafStateZ = z.record(z.string(), z.unknown());
export type LeafState = z.infer<typeof leafStateZ>;

/** Captures a single `afterUpdate` invocation on a {@link Leaf}, including the
 * state at that point and the state immediately preceding it. */
export interface UpdateCall {
  state: LeafState;
  prevState: LeafState;
}

/** Bundled aether `Leaf` stub for tests. Registered in a test registry under any type
 * name; records every lifecycle invocation for later assertion. The schema accepts any
 * record-shaped state. */
export class Leaf extends aether.Leaf<typeof leafStateZ> {
  static readonly TYPE: string = "aetherTest.Leaf";
  static readonly stateZ = leafStateZ;
  schema = Leaf.stateZ;

  /** Every call to {@link afterUpdate}, in order. */
  readonly updateCalls: UpdateCall[] = [];
  /** Number of `afterDelete` invocations. */
  deleteCallCount = 0;

  afterUpdate(): void {
    this.updateCalls.push({ state: this.state, prevState: this.prevState });
  }

  afterDelete(): void {
    this.deleteCallCount += 1;
  }
}
