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
import { StringValue } from "@/vis/stringValue";
import { stringValue } from "@/vis/stringValue/aether";

// Stands in for the worker component and reports its source as stale.
class StaleStringValue extends aether.Leaf<typeof stringValue.stateZ> {
  static readonly TYPE = stringValue.StringValue.TYPE;
  override schema = stringValue.stateZ;

  override afterUpdate(): void {
    if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
  }
}

describe("StringValue.use", () => {
  it("should return the staleness the worker reports", async () => {
    const { result } = renderHook(
      () => StringValue.use({ aetherKey: "stringValue", box: box.ZERO }),
      {
        registry: { [StaleStringValue.TYPE]: StaleStringValue },
      },
    );
    await waitFor(() => expect(result.current.stale).toBe(true));
  });
});
