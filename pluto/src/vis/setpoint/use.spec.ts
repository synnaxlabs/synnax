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
import { Setpoint } from "@/vis/setpoint";
import { setpoint } from "@/vis/setpoint/aether";

describe("Setpoint.use", () => {
  it("should render once for each prop change", async () => {
    let renders = 0;
    const { result, rerender } = renderHook(
      ({ sink }: { sink: telem.NumberSinkSpec }) => {
        renders++;
        return Setpoint.use({ aetherKey: "setpoint", sink });
      },
      {
        rtl: { initialProps: { sink: telem.noopNumericSinkSpec } },
        registry: setpoint.REGISTRY,
      },
    );
    // The noop source reports 0 after mount, which renders once more.
    await waitFor(() => expect(result.current.value).toBe(0));
    renders = 0;
    rerender({ sink: { ...telem.noopNumericSinkSpec, props: { key: 1 } } });
    expect(renders).toBe(1);
  });

  it("should follow the value the source reports", async () => {
    const source = telemTest.source(3);
    const { result } = renderHook(
      () =>
        Setpoint.use({
          aetherKey: "setpoint",
          sink: telem.noopNumericSinkSpec,
          source: telemTest.numberSourceSpec(source),
        }),
      { registry: setpoint.REGISTRY },
    );
    await waitFor(() => expect(result.current.value).toBe(3));
    source.setValue(5);
    await waitFor(() => expect(result.current.value).toBe(5));
  });
});
