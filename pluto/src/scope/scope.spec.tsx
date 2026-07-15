// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useState,
} from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Scope } from "@/scope";

interface SelectParams {
  key: string;
  suffix?: string;
}

const select = ({ key, suffix = "" }: SelectParams): string => `${key}${suffix}`;

const selector: Scope.SelectorHooks<SelectArgs, string> = [
  (args) => select(args),
  () => useCallback((args: SelectArgs) => select(args), []),
];

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

  describe("bindSelector", () => {
    let s: Scope.Instance<string>;
    beforeEach(() => {
      s = Scope.create<string>("Test");
    });
    const provider = (value: string) => {
      const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <s.Provider value={value}>{children}</s.Provider>
      );
      return Wrapper;
    };

    describe("reactive hook", () => {
      it("should source the key from the provider", () => {
        const [useSelect] = s.bindSelector(selector);
        const { result } = renderHook(() => useSelect(), {
          wrapper: provider("ctx"),
        });
        expect(result.current).toEqual("ctx");
      });

      it("should let an explicit key override the provider", () => {
        const [useSelect] = s.bindSelector(selector);
        const { result } = renderHook(() => useSelect({ key: "explicit" }), {
          wrapper: provider("ctx"),
        });
        expect(result.current).toEqual("explicit");
      });

      it("should forward non-key arguments unchanged", () => {
        const [useSelect] = s.bindSelector(selector);
        const { result } = renderHook(() => useSelect({ suffix: "!" }), {
          wrapper: provider("ctx"),
        });
        expect(result.current).toEqual("ctx!");
      });

      it("should throw when no key or provider is present", () => {
        const [useSelect] = s.bindSelector(selector);
        expect(() => renderHook(() => useSelect())).toThrow(
          "Test scope requires a key",
        );
      });
    });

    describe("getter hook", () => {
      it("should inject the scope key into each read", () => {
        const [, useGet] = s.bindSelector(selector);
        const { result } = renderHook(() => useGet(), { wrapper: provider("ctx") });
        expect(result.current()).toEqual("ctx");
      });

      it("should let an explicit key override the scope key", () => {
        const [, useGet] = s.bindSelector(selector);
        const { result } = renderHook(() => useGet(), { wrapper: provider("ctx") });
        expect(result.current({ key: "explicit" })).toEqual("explicit");
      });

      it("should forward non-key arguments unchanged", () => {
        const [, useGet] = s.bindSelector(selector);
        const { result } = renderHook(() => useGet(), { wrapper: provider("ctx") });
        expect(result.current({ suffix: "!" })).toEqual("ctx!");
      });

      it("should read an undefined key when no key or provider is present", () => {
        const [, useGet] = s.bindSelector(selector);
        const { result } = renderHook(() => useGet());
        expect(result.current()).toEqual("undefined");
      });

      it("should return a stable getter while the scope key is unchanged", () => {
        const [, useGet] = s.bindSelector(selector);
        const { result, rerender } = renderHook(() => useGet(), {
          wrapper: provider("ctx"),
        });
        const first = result.current;
        rerender();
        expect(result.current).toBe(first);
      });

      it("should return a fresh getter when the scope key changes", () => {
        const [, useGet] = s.bindSelector(selector);
        let setKey: (key: string) => void = () => {};
        const wrapper = ({ children }: PropsWithChildren): ReactElement => {
          const [key, set] = useState("a");
          setKey = set;
          return <s.Provider value={key}>{children}</s.Provider>;
        };
        const { result } = renderHook(() => useGet(), { wrapper });
        const first = result.current;
        expect(first()).toEqual("a");
        act(() => setKey("b"));
        expect(result.current).not.toBe(first);
        expect(result.current()).toEqual("b");
      });
    });
  });
});
