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
import { Scale } from "@/vis/scale";
import { scale } from "@/vis/scale/aether";

// Stands in for the worker component and reports its source as stale.
class StaleScale extends aether.Leaf<typeof scale.Scale.z> {
  static readonly TYPE = scale.Scale.TYPE;
  schema = scale.Scale.z;

  afterUpdate(): void {
    if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
  }
}

describe("Scale.use", () => {
  it("should return the staleness the worker reports", async () => {
    const { result } = renderHook(
      () => Scale.use({ aetherKey: "scale", box: box.ZERO }),
      {
        registry: { [StaleScale.TYPE]: StaleScale },
      },
    );
    await waitFor(() => expect(result.current.stale).toBe(true));
  });
});
