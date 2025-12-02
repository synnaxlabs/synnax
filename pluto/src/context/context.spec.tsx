// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { createRequired } from "@/context/context";

describe("context.createRequired", () => {
  it("should create a context with the correct display name", () => {
    const [ctx] = createRequired<string>("TestContext", "TestProvider");
    expect(ctx.displayName).toBe("TestContext");
  });

  it("should return the context value when used inside a provider", () => {
    const [Context, useContext] = createRequired<string>("TestContext", "TestProvider");
    const wrapper = ({ children }: PropsWithChildren) => (
      <Context value="test-value">{children}</Context>
    );

    const { result } = renderHook(() => useContext("useTest"), { wrapper });
    expect(result.current).toBe("test-value");
  });

  it("should throw an error when used outside a provider", () => {
    const [, useContext] = createRequired<string>("TestContext", "TestProvider");

    expect(() => {
      renderHook(() => useContext("useTest"));
    }).toThrow("useTest must be used within TestProvider");
  });

  it("should include the hook name in the error message", () => {
    const [, useContext] = createRequired<string>("TestContext", "TestProvider");

    expect(() => {
      renderHook(() => useContext("useMyCustomHook"));
    }).toThrow("useMyCustomHook must be used within TestProvider");
  });

  it("should include the provider name in the error message", () => {
    const [, useContext] = createRequired<string>("TestContext", "MyCustomProvider");

    expect(() => {
      renderHook(() => useContext("useTest"));
    }).toThrow("useTest must be used within MyCustomProvider");
  });

  it("should work with object context values", () => {
    interface TestValue {
      name: string;
      count: number;
    }
    const [Context, useContext] = createRequired<TestValue>(
      "TestContext",
      "TestProvider",
    );
    const testValue: TestValue = { name: "test", count: 42 };
    const wrapper = ({ children }: PropsWithChildren) => (
      <Context value={testValue}>{children}</Context>
    );

    const { result } = renderHook(() => useContext("useTest"), { wrapper });
    expect(result.current).toEqual({ name: "test", count: 42 });
  });

  it("should work with function context values", () => {
    const [Context, useContext] = createRequired<() => number>(
      "TestContext",
      "TestProvider",
    );
    const testFn = () => 42;
    const wrapper = ({ children }: PropsWithChildren) => (
      <Context value={testFn}>{children}</Context>
    );

    const { result } = renderHook(() => useContext("useTest"), { wrapper });
    expect(result.current()).toBe(42);
  });

  it("should work with different hook names for the same context", () => {
    const [Context, useContext] = createRequired<string>("TestContext", "TestProvider");
    const wrapper = ({ children }: PropsWithChildren) => (
      <Context value="value">{children}</Context>
    );

    const { result: result1 } = renderHook(() => useContext("useFirst"), { wrapper });
    const { result: result2 } = renderHook(() => useContext("useSecond"), { wrapper });

    expect(result1.current).toBe("value");
    expect(result2.current).toBe("value");
  });

  it("should throw different error messages for different hook names", () => {
    const [, useContext] = createRequired<string>("TestContext", "TestProvider");

    expect(() => {
      renderHook(() => useContext("useFirst"));
    }).toThrow("useFirst must be used within TestProvider");

    expect(() => {
      renderHook(() => useContext("useSecond"));
    }).toThrow("useSecond must be used within TestProvider");
  });
});
