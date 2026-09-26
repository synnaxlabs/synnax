// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { Store } from "@/store";

const Presence = Store.createPresence("test");

const staticWrapper = (value: string | undefined) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Presence.Context value={value}>{children}</Presence.Context>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("Store.createPresence", () => {
  describe("useIsPresent", () => {
    it("should be true for the single present key", () => {
      const present = renderHook(() => Presence.useIsPresent("b"), {
        wrapper: staticWrapper("b"),
      });
      const absent = renderHook(() => Presence.useIsPresent("a"), {
        wrapper: staticWrapper("b"),
      });
      expect(present.result.current).toBe(true);
      expect(absent.result.current).toBe(false);
    });

    it("should be false when no key is present", () => {
      const { result } = renderHook(() => Presence.useIsPresent("a"), {
        wrapper: staticWrapper(undefined),
      });
      expect(result.current).toBe(false);
    });

    it("should be false when used outside a provider", () => {
      const { result } = renderHook(() => Presence.useIsPresent("a"));
      expect(result.current).toBe(false);
    });

    it("should re-render a key only when it gains or loses presence", () => {
      let renders = 0;
      let setValue: (key: string) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState("b");
        setValue = setState;
        return <Presence.Context value={value}>{children}</Presence.Context>;
      };
      const { result } = renderHook(
        () => {
          renders++;
          return Presence.useIsPresent("a");
        },
        { wrapper },
      );
      expect(result.current).toBe(false);
      const baseline = renders;
      act(() => setValue("c"));
      expect(renders).toBe(baseline);
      act(() => setValue("a"));
      expect(result.current).toBe(true);
      expect(renders).toBeGreaterThan(baseline);
    });
  });

  describe("usePresent", () => {
    it("should return the present key", () => {
      const { result } = renderHook(() => Presence.usePresent(), {
        wrapper: staticWrapper("b"),
      });
      expect(result.current).toEqual("b");
    });

    it("should track presence changes", () => {
      let setValue: (key: string | undefined) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState<string | undefined>("a");
        setValue = setState;
        return <Presence.Context value={value}>{children}</Presence.Context>;
      };
      const { result } = renderHook(() => Presence.usePresent(), { wrapper });
      expect(result.current).toEqual("a");
      act(() => setValue("b"));
      expect(result.current).toEqual("b");
      act(() => setValue(undefined));
      expect(result.current).toBeUndefined();
    });
  });
});
