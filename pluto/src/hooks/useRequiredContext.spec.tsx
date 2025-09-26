// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { renderHook } from "@testing-library/react";
import { createContext } from "react";
import { describe, expect, it } from "vitest";

import { useRequiredContext } from "@/hooks/useRequiredContext";

interface TestContextValue {
  value: string;
  count: number;
}

describe("useRequiredContext", () => {
  it("should return the context value when it is not null", () => {
    const TestContext = createContext<TestContextValue | null>({
      value: "test",
      count: 42,
    });
    TestContext.displayName = "TestContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <TestContext.Provider value={{ value: "test", count: 42 }}>
        {children}
      </TestContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(TestContext), {
      wrapper,
    });

    expect(result.current).toEqual({ value: "test", count: 42 });
    expect(result.current.value).toBe("test");
    expect(result.current.count).toBe(42);
  });

  it("should throw NotFoundError when context value is null", () => {
    const TestContext = createContext<TestContextValue | null>(null);
    TestContext.displayName = "TestContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <TestContext.Provider value={null}>{children}</TestContext.Provider>
    );

    expect(() =>
      renderHook(() => useRequiredContext(TestContext), { wrapper }),
    ).toThrow(NotFoundError);

    expect(() =>
      renderHook(() => useRequiredContext(TestContext), { wrapper }),
    ).toThrow("useRequiredContext: context value is null for TestContext");
  });

  it("should throw NotFoundError when context value is undefined", () => {
    const TestContext = createContext<TestContextValue | null>(null);
    TestContext.displayName = "UndefinedTestContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <TestContext.Provider value={undefined as any}>{children}</TestContext.Provider>
    );

    expect(() =>
      renderHook(() => useRequiredContext(TestContext), { wrapper }),
    ).toThrow(NotFoundError);

    expect(() =>
      renderHook(() => useRequiredContext(TestContext), { wrapper }),
    ).toThrow("useRequiredContext: context value is null for UndefinedTestContext");
  });

  it("should throw when used outside of a provider", () => {
    const TestContext = createContext<TestContextValue | null>(null);
    TestContext.displayName = "NoProviderContext";

    expect(() => renderHook(() => useRequiredContext(TestContext))).toThrow(
      NotFoundError,
    );

    expect(() => renderHook(() => useRequiredContext(TestContext))).toThrow(
      "useRequiredContext: context value is null for NoProviderContext",
    );
  });

  it("should work with primitive context values", () => {
    const StringContext = createContext<string | null>("default");
    StringContext.displayName = "StringContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <StringContext.Provider value="test string">{children}</StringContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(StringContext), {
      wrapper,
    });

    expect(result.current).toBe("test string");
  });

  it("should work with number context values including zero", () => {
    const NumberContext = createContext<number | null>(null);
    NumberContext.displayName = "NumberContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <NumberContext.Provider value={0}>{children}</NumberContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(NumberContext), {
      wrapper,
    });

    expect(result.current).toBe(0);
  });

  it("should work with boolean false values", () => {
    const BooleanContext = createContext<boolean | null>(null);
    BooleanContext.displayName = "BooleanContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <BooleanContext.Provider value={false}>{children}</BooleanContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(BooleanContext), {
      wrapper,
    });

    expect(result.current).toBe(false);
  });

  it("should work with empty string values", () => {
    const EmptyStringContext = createContext<string | null>(null);
    EmptyStringContext.displayName = "EmptyStringContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EmptyStringContext.Provider value="">{children}</EmptyStringContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(EmptyStringContext), {
      wrapper,
    });

    expect(result.current).toBe("");
  });

  it("should work with array context values", () => {
    const ArrayContext = createContext<number[] | null>(null);
    ArrayContext.displayName = "ArrayContext";

    const testArray = [1, 2, 3];
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <ArrayContext.Provider value={testArray}>{children}</ArrayContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(ArrayContext), {
      wrapper,
    });

    expect(result.current).toEqual([1, 2, 3]);
    expect(result.current).toBe(testArray);
  });

  it("should work with empty array values", () => {
    const EmptyArrayContext = createContext<any[] | null>(null);
    EmptyArrayContext.displayName = "EmptyArrayContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <EmptyArrayContext.Provider value={[]}>{children}</EmptyArrayContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(EmptyArrayContext), {
      wrapper,
    });

    expect(result.current).toEqual([]);
  });

  it("should handle context without displayName", () => {
    const NoNameContext = createContext<string | null>(null);

    expect(() => renderHook(() => useRequiredContext(NoNameContext))).toThrow(
      "useRequiredContext: context value is null for undefined",
    );
  });

  it("should update when context value changes", () => {
    const DynamicContext = createContext<number | null>(null);
    DynamicContext.displayName = "DynamicContext";

    let value = 1;
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <DynamicContext.Provider value={value}>{children}</DynamicContext.Provider>
    );

    const { result, rerender } = renderHook(() => useRequiredContext(DynamicContext), {
      wrapper,
    });

    expect(result.current).toBe(1);

    value = 2;
    rerender();

    expect(result.current).toBe(2);
  });

  it("should work with nested context providers", () => {
    const OuterContext = createContext<string | null>(null);
    const InnerContext = createContext<number | null>(null);
    OuterContext.displayName = "OuterContext";
    InnerContext.displayName = "InnerContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <OuterContext.Provider value="outer">
        <InnerContext.Provider value={42}>{children}</InnerContext.Provider>
      </OuterContext.Provider>
    );

    const { result } = renderHook(
      () => ({
        outer: useRequiredContext(OuterContext),
        inner: useRequiredContext(InnerContext),
      }),
      { wrapper },
    );

    expect(result.current.outer).toBe("outer");
    expect(result.current.inner).toBe(42);
  });

  it("should properly type the return value as NonNullable", () => {
    const TypedContext = createContext<{ optional?: string } | null>(null);
    TypedContext.displayName = "TypedContext";

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <TypedContext.Provider value={{ optional: "value" }}>
        {children}
      </TypedContext.Provider>
    );

    const { result } = renderHook(() => useRequiredContext(TypedContext), {
      wrapper,
    });

    type ResultType = typeof result.current;
    const _typeCheck: ResultType = { optional: "test" };
    expect(result.current.optional).toBe("value");
  });
});
