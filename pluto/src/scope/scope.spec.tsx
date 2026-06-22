// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Scope } from "@/scope";

interface SelectArgs {
  key: string;
  suffix?: string;
}

const select = ({ key, suffix = "" }: SelectArgs): string => `${key}${suffix}`;

describe("Scope", () => {
  describe("use", () => {
    it("should return the override when no provider is present", () => {
      const s = Scope.create<string>("Test");
      const { result } = renderHook(() => s.use("override"));
      expect(result.current).toEqual("override");
    });

    it("should throw when neither an override nor a provider is present", () => {
      const s = Scope.create<string>("Test");
      expect(() => renderHook(() => s.use())).toThrow("Test scope requires a key");
    });

    it("should resolve the key from the surrounding provider", () => {
      const s = Scope.create<string>("Test");
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <s.Provider value="from-context">{children}</s.Provider>
      );
      const { result } = renderHook(() => s.use(), { wrapper });
      expect(result.current).toEqual("from-context");
    });

    it("should let an explicit override win over the provider", () => {
      const s = Scope.create<string>("Test");
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <s.Provider value="from-context">{children}</s.Provider>
      );
      const { result } = renderHook(() => s.use("override"), { wrapper });
      expect(result.current).toEqual("override");
    });
  });

  describe("useOptional", () => {
    it("should return undefined when no provider or override is present", () => {
      const s = Scope.create<string>("Test");
      const { result } = renderHook(() => s.useOptional());
      expect(result.current).toBeUndefined();
    });
  });

  describe("bindHook", () => {
    it("should source the key from the provider", () => {
      const s = Scope.create<string>("Test");
      const useSelect = s.bindHook(select);
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <s.Provider value="ctx">{children}</s.Provider>
      );
      const { result } = renderHook(() => useSelect(), { wrapper });
      expect(result.current).toEqual("ctx");
    });

    it("should forward non-key arguments unchanged", () => {
      const s = Scope.create<string>("Test");
      const useSelect = s.bindHook(select);
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <s.Provider value="ctx">{children}</s.Provider>
      );
      const { result } = renderHook(() => useSelect({ suffix: "!" }), { wrapper });
      expect(result.current).toEqual("ctx!");
    });

    it("should let an explicit key override the provider", () => {
      const s = Scope.create<string>("Test");
      const useSelect = s.bindHook(select);
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <s.Provider value="ctx">{children}</s.Provider>
      );
      const { result } = renderHook(() => useSelect({ key: "explicit" }), { wrapper });
      expect(result.current).toEqual("explicit");
    });
  });
});
