// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { createProvider } from "@/aether/test/provider";

const testStateZ = z.object({
  value: z.number(),
});

class TestLeaf extends aether.Leaf<typeof testStateZ> {
  static readonly TYPE = "TestLeaf";
  schema = testStateZ;
  updateFn = vi.fn();

  afterUpdate(): void {
    this.updateFn();
  }

  afterDelete(): void {}
}

describe("createProvider", () => {
  it("should return a functional component", () => {
    const Provider = createProvider({});
    expect(typeof Provider).toBe("function");
  });

  it("should render children", () => {
    const Provider = createProvider({});
    const { getByText } = render(
      <Provider>
        <div>Test Child</div>
      </Provider>,
    );
    expect(getByText("Test Child")).toBeDefined();
  });

  it("should allow Aether.use with registered components", async () => {
    const Provider = createProvider({
      [TestLeaf.TYPE]: TestLeaf,
    });

    const TestComponent: FC = () => {
      const [, state] = Aether.use({
        type: TestLeaf.TYPE,
        aetherKey: "test-key",
        schema: testStateZ,
        initialState: { value: 42 },
      });
      return <div data-testid="value">{state.value}</div>;
    };

    const { getByTestId } = render(
      <Provider>
        <TestComponent />
      </Provider>,
    );

    await expect.poll(() => getByTestId("value").textContent).toBe("42");
  });

  it("should create isolated instances for each call", () => {
    const Provider1 = createProvider({});
    const Provider2 = createProvider({});

    expect(Provider1).not.toBe(Provider2);
  });

  it("should work with renderHook", () => {
    const Provider = createProvider({
      [TestLeaf.TYPE]: TestLeaf,
    });

    const wrapper: FC<PropsWithChildren> = ({ children }) => (
      <Provider>{children}</Provider>
    );

    const { result } = renderHook(
      () =>
        Aether.use({
          type: TestLeaf.TYPE,
          aetherKey: "hook-test",
          schema: testStateZ,
          initialState: { value: 99 },
        }),
      { wrapper },
    );

    expect(result.current[1].value).toBe(99);
  });
});
