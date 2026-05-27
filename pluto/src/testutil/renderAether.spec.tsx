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

import { Aether } from "@/aether";
import { TestLeaf } from "@/aether/test/TestLeaf";
import { alamos } from "@/alamos/aether";
import { renderAether } from "@/testutil/renderAether";
import { theming } from "@/theming/aether";

const HookComponent = ({ initialValue }: { initialValue: number }) => {
  Aether.use({
    type: TestLeaf.TYPE,
    schema: TestLeaf.stateZ,
    aetherKey: "consumer",
    initialState: { value: initialValue },
  });
  return <div>hook-component</div>;
};

describe("renderAether", () => {
  it("renders the UI inside the wrapper", () => {
    const { getByText } = renderAether(<div>hello</div>);
    expect(getByText("hello")).toBeDefined();
  });

  it("exposes the worker-side providers via the result", () => {
    const { providers } = renderAether(<div />);
    expect(providers.alamos).toBeInstanceOf(alamos.Provider);
    expect(providers.theming).toBeInstanceOf(theming.Provider);
  });

  it("mounts components used via Aether.use under the telem provider path", async () => {
    const { root } = renderAether(<HookComponent initialValue={7} />);
    await waitFor(() => {
      const leaf = root.findChildAtPath([
        "alamos",
        "status",
        "synnax",
        "theming",
        "telem",
        "consumer",
      ]);
      expect(leaf).toBeInstanceOf(TestLeaf);
      expect((leaf as TestLeaf).state).toEqual({ value: 7 });
    });
  });

  it("applies provider overrides through the stack", () => {
    const { providers } = renderAether(<div />, {
      providers: { alamos: { level: "debug" } },
    });
    expect(providers.alamos.state.level).toBe("debug");
  });
});
