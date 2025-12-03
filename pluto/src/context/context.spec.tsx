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

import { create } from "@/context/context";

describe("create", () => {
  describe("with a default value", () => {
    it("should create a context with the correct display name", () => {
      const [Context] = create({
        defaultValue: "test-value",
        displayName: "TestContext",
      });
      expect(Context.displayName).toBe("TestContext");
    });

    it("should return the default value when used outside a provider", () => {
      const [_, useContext] = create({
        defaultValue: "test-value",
        displayName: "TestContext",
      });
      const { result } = renderHook(useContext);
      expect(result.current).toBe("test-value");
    });
    it("should return the context value when used inside a provider", () => {
      const [Context, useContext] = create({
        defaultValue: "test-value",
        displayName: "TestContext",
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Context value="updated-value">{children}</Context>
      );
      const { result } = renderHook(useContext, { wrapper });
      expect(result.current).toBe("updated-value");
    });
  });
  describe("without a default value", () => {
    it("should create a context with the correct display name", () => {
      const [Context] = create({
        displayName: "TestContext",
        providerName: "TestProvider",
      });
      expect(Context.displayName).toBe("TestContext");
    });

    it("should return the context value when used inside a provider", () => {
      const [Context, useContext] = create<string>({
        displayName: "TestContext",
        providerName: "TestProvider",
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Context value="test-value">{children}</Context>
      );
      const { result } = renderHook(() => useContext(""), { wrapper });
      expect(result.current).toBe("test-value");
    });
    it("should throw an error containing the hook name and provider name when used outside a provider", () => {
      const [, useContext] = create<string>({
        displayName: "TestContext",
        providerName: "TestProvider",
      });
      expect(() => {
        renderHook(() => useContext("useTest"));
      }).toThrow("useTest must be used within TestProvider");
    });

    it("should work with different hook names for the same context", () => {
      const [Context, useContext] = create<string>({
        displayName: "TestContext",
        providerName: "TestProvider",
      });
      const wrapper = ({ children }: PropsWithChildren) => (
        <Context value="value">{children}</Context>
      );
      const { result: result1 } = renderHook(() => useContext("useFirst"), { wrapper });
      const { result: result2 } = renderHook(() => useContext("useSecond"), {
        wrapper,
      });
      expect(result1.current).toBe("value");
      expect(result2.current).toBe("value");
    });

    it("should throw different error messages for different hook names", () => {
      const [, useContext] = create<string>({
        displayName: "TestContext",
        providerName: "TestProvider",
      });
      expect(() => {
        renderHook(() => useContext("useFirst"));
      }).toThrow("useFirst must be used within TestProvider");
      expect(() => {
        renderHook(() => useContext("useSecond"));
      }).toThrow("useSecond must be used within TestProvider");
    });
  });
});
