// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { renderHook } from "@/testutil";
import { SymbolNode } from "@/vis/symbolNode";
import { symbolNode } from "@/vis/symbolNode/aether";

describe("SymbolNode.use", () => {
  it("should follow the reading of each source", async () => {
    const source = telemTest.source<number>(7);
    const { result } = renderHook(
      () =>
        SymbolNode.use({
          aetherKey: "symbol",
          sources: { active: telemTest.numberSourceSpec(source) },
          sinks: {},
        }),
      { registry: symbolNode.REGISTRY },
    );
    await waitFor(() =>
      expect(result.current.readings).toEqual({ active: { value: 7, stale: false } }),
    );
    source.setValue(5);
    await waitFor(() =>
      expect(result.current.readings).toEqual({ active: { value: 5, stale: false } }),
    );
  });

  it("should render once for each prop change", async () => {
    let renders = 0;
    const { result, rerender } = renderHook(
      ({ sink }: { sink: telem.NumberSinkSpec }) => {
        renders++;
        return SymbolNode.use({
          aetherKey: "symbol",
          sources: { active: telem.noopNumericSourceSpec },
          sinks: { active: sink },
        });
      },
      {
        rtl: { initialProps: { sink: telem.noopNumericSinkSpec } },
        registry: symbolNode.REGISTRY,
      },
    );
    await waitFor(() => expect(result.current.readings.active).toBeDefined());
    renders = 0;
    rerender({ sink: { ...telem.noopNumericSinkSpec, props: { key: 1 } } });
    expect(renders).toBe(1);
  });
});
