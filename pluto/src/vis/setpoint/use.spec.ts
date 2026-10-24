// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { telem } from "@/telem/aether";
import { renderHook } from "@/testutil/render";
import { Setpoint } from "@/vis/setpoint";
import { setpoint } from "@/vis/setpoint/aether";

describe("Setpoint.use", () => {
  it("should render once for each prop change", () => {
    let renders = 0;
    const { rerender } = renderHook(
      ({ sink }: { sink: telem.NumberSinkSpec }) => {
        renders++;
        return Setpoint.use({ aetherKey: "setpoint", sink });
      },
      {
        rtl: { initialProps: { sink: telem.noopNumericSinkSpec } },
        registry: setpoint.REGISTRY,
      },
    );
    expect(renders).toBe(1);
    rerender({ sink: { ...telem.noopNumericSinkSpec, props: { key: 1 } } });
    expect(renders).toBe(2);
  });
});
