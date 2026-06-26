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
import { aetherTest } from "@/aether/test";
import { alamos } from "@/alamos/aether";
import { render } from "@/testutil/render";
import { theming } from "@/theming/aether";

const HookComponent = ({ initialValue }: { initialValue: number }) => {
  Aether.use({
    type: aetherTest.TestLeaf.TYPE,
    schema: aetherTest.TestLeaf.stateZ,
    aetherKey: "consumer",
    initialState: { value: initialValue },
  });
  return <div>hook-component</div>;
};

describe("render", () => {
  it("renders the UI inside the wrapper", () => {
    const { getByText } = render(<div>hello</div>);
    expect(getByText("hello")).toBeDefined();
  });

  it("exposes the worker-side providers via the result", () => {
    const { providers } = render(<div />);
    expect(providers.alamos).toBeInstanceOf(alamos.Provider);
    expect(providers.theming).toBeInstanceOf(theming.Provider);
  });

  it("mounts components used via Aether.use under the telem provider path", async () => {
    const { root } = render(<HookComponent initialValue={7} />);
    await waitFor(() => {
      const leaf = root.findChildAtPath([
        "alamos",
        "status",
        "synnax",
        "theming",
        "telem",
        "consumer",
      ]);
      expect(leaf).toBeInstanceOf(aetherTest.TestLeaf);
      expect((leaf as aetherTest.TestLeaf).state).toEqual({ value: 7 });
    });
  });

  it("applies provider overrides through the stack", () => {
    const { providers } = render(<div />, { alamos: { level: "debug" } });
    expect(providers.alamos?.state.level).toBe("debug");
  });

  it("drops a provider from the stack when toggled off", () => {
    const { providers } = render(<div />, { telem: false });
    expect(providers.telem).toBeNull();
  });
});
