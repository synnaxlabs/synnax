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
import { Gauge } from "@/vis/gauge";
import { gauge } from "@/vis/gauge/aether";

// Stands in for the worker component and reports its source as stale.
class StaleGauge extends aether.Leaf<typeof gauge.Gauge.z> {
  static readonly TYPE = gauge.Gauge.TYPE;
  override schema = gauge.Gauge.z;

  override afterUpdate(): void {
    if (!this.state.stale) this.setState((s) => ({ ...s, stale: true }));
  }
}

describe("Gauge.use", () => {
  it("should render once for each prop change", () => {
    let renders = 0;
    const { rerender } = renderHook(
      ({ precision }: { precision: number }) => {
        renders++;
        Gauge.use({ aetherKey: "gauge", box: box.ZERO, precision });
      },
      {
        rtl: { initialProps: { precision: 2 } },
        render: true,
        registry: gauge.REGISTRY,
      },
    );
    expect(renders).toBe(1);
    rerender({ precision: 3 });
    expect(renders).toBe(2);
  });

  it("should return the staleness the worker reports", async () => {
    const { result } = renderHook(
      () => Gauge.use({ aetherKey: "gauge", box: box.ZERO }),
      {
        registry: { [StaleGauge.TYPE]: StaleGauge },
      },
    );
    await waitFor(() => expect(result.current.stale).toBe(true));
  });
});
