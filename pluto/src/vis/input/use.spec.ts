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
import { Input } from "@/vis/input";
import { input } from "@/vis/input/aether";

describe("Input.use", () => {
  it("should render once for each prop change", () => {
    let renders = 0;
    const { rerender } = renderHook(
      ({ sink }: { sink: telem.StringSinkSpec }) => {
        renders++;
        return Input.use({
          aetherKey: "input",
          sink,
          source: telem.noopStringSourceSpec,
        });
      },
      {
        rtl: { initialProps: { sink: telem.noopStringSinkSpec } },
        registry: input.REGISTRY,
      },
    );
    expect(renders).toBe(1);
    rerender({ sink: { ...telem.noopStringSinkSpec, props: { key: 1 } } });
    expect(renders).toBe(2);
  });

  it("should follow the value the source reports", async () => {
    const source = telemTest.source("closed");
    const { result } = renderHook(
      () =>
        Input.use({
          aetherKey: "input",
          sink: telem.noopStringSinkSpec,
          source: telemTest.stringSourceSpec(source),
        }),
      { registry: input.REGISTRY },
    );
    await waitFor(() => expect(result.current.value).toBe("closed"));
    source.setValue("open");
    await waitFor(() => expect(result.current.value).toBe("open"));
  });
});
