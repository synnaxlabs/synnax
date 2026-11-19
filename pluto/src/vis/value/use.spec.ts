// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { aether } from "@/aether/aether";
import { renderHook } from "@/testutil/render";
import { Value } from "@/vis/value";
import { value } from "@/vis/value/aether";

// Stands in for the worker component and reports its source as stale.
class StaleValue extends aether.Leaf<typeof value.Value.z> {
  static readonly TYPE = value.Value.TYPE;
  schema = value.Value.z;

  afterUpdate(): void {
    if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
  }
}

describe("Value.use", () => {
  it("should return the staleness the worker reports", async () => {
    const { result } = renderHook(
      () => Value.use({ aetherKey: "value", box: box.ZERO }),
      {
        registry: { [StaleValue.TYPE]: StaleValue },
      },
    );
    await waitFor(() => expect(result.current.stale).toBe(true));
  });
});
