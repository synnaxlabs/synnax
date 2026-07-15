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
import { type UpdateCall } from "@/aether/test/Leaf";

export const compositeStateZ = z.record(z.string(), z.unknown());
export type CompositeState = z.infer<typeof compositeStateZ>;

/** Bundled aether `Composite` stub for tests. Same shape as {@link Leaf} but
 * accepts arbitrary descendants — drop it into a registry to stand in for a real
 * Composite parent while keeping its children under the harness's normal child
 * registry. */
export class Composite extends aether.Composite<typeof compositeStateZ> {
  static readonly TYPE: string = "aetherTest.Composite";
  static readonly stateZ = compositeStateZ;
  schema = Composite.stateZ;

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
