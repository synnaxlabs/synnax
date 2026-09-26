// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Store } from "@/store";

const Set = Store.createMembership("test");

const staticWrapper = (
  value: Store.MembershipValue<string>,
  onItem?: (key: record.Key) => void,
) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Set.Context value={value} onItem={onItem}>
      {children}
    </Set.Context>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("Store.createMembership", () => {
  describe("useIsMember", () => {
    it("should be true for the item matching a single value", () => {
      const { result } = renderHook(() => Set.useIsMember("a"), {
        wrapper: staticWrapper("a"),
      });
      expect(result.current).toBe(true);
    });

    it("should be false for an item not in the value", () => {
      const { result } = renderHook(() => Set.useIsMember("b"), {
        wrapper: staticWrapper("a"),
      });
      expect(result.current).toBe(false);
    });

    it("should be true for every item included in an array value", () => {
      const inSet = renderHook(() => Set.useIsMember("c"), {
        wrapper: staticWrapper(["a", "c"]),
      });
      const notInSet = renderHook(() => Set.useIsMember("b"), {
        wrapper: staticWrapper(["a", "c"]),
      });
      expect(inSet.result.current).toBe(true);
      expect(notInSet.result.current).toBe(false);
    });

    it("should be false when used outside a provider", () => {
      const { result } = renderHook(() => Set.useIsMember("a"));
      expect(result.current).toBe(false);
    });

    it("should re-render an item only when its own membership flips", () => {
      let renders = 0;
      let setValue: (value: string[]) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState<string[]>(["a"]);
        setValue = setState;
        return <Set.Context value={value}>{children}</Set.Context>;
      };
      renderHook(
        () => {
          renders++;
          return Set.useIsMember("a");
        },
        { wrapper },
      );
      const baseline = renders;
      act(() => setValue(["a", "b"]));
      expect(renders).toBe(baseline);
      act(() => setValue(["b"]));
      expect(renders).toBeGreaterThan(baseline);
    });
  });

  describe("useItem", () => {
    it("should call the provider's onItem with the item's key", () => {
      const onItem = vi.fn();
      const { result } = renderHook(() => Set.useItem("b"), {
        wrapper: staticWrapper("a", onItem),
      });
      act(() => result.current.onItem());
      expect(onItem).toHaveBeenCalledTimes(1);
      expect(onItem).toHaveBeenCalledWith("b");
    });
  });

  describe("useMembers", () => {
    it("should return a single value as a one-element array", () => {
      const { result } = renderHook(() => Set.useMembers(), {
        wrapper: staticWrapper("a"),
      });
      expect(result.current).toEqual(["a"]);
    });

    it("should return an empty array when the set is empty", () => {
      const { result } = renderHook(() => Set.useMembers(), {
        wrapper: staticWrapper(undefined),
      });
      expect(result.current).toEqual([]);
    });

    it("should track membership changes", () => {
      let setValue: (value: string[]) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState<string[]>(["a"]);
        setValue = setState;
        return <Set.Context value={value}>{children}</Set.Context>;
      };
      const { result } = renderHook(() => Set.useMembers(), { wrapper });
      expect(result.current).toEqual(["a"]);
      act(() => setValue(["a", "b"]));
      expect(result.current).toEqual(["a", "b"]);
    });
  });

  describe("useMemberAmong", () => {
    it("should return the member among the given keys", () => {
      const { result } = renderHook(() => Set.useMemberAmong(["b", "c"]), {
        wrapper: staticWrapper(["a", "c"]),
      });
      expect(result.current).toEqual("c");
    });

    it("should return undefined when none of the keys is a member", () => {
      const { result } = renderHook(() => Set.useMemberAmong(["b"]), {
        wrapper: staticWrapper(["a"]),
      });
      expect(result.current).toBeUndefined();
    });

    it("should prefer the earliest key in the set's order", () => {
      const { result } = renderHook(() => Set.useMemberAmong(["a", "b"]), {
        wrapper: staticWrapper(["b", "a"]),
      });
      expect(result.current).toEqual("b");
    });

    it("should not re-render on changes outside the given keys", () => {
      let renders = 0;
      let setValue: (value: string[]) => void = () => {};
      const wrapper = ({ children }: PropsWithChildren): ReactElement => {
        const [value, setState] = useState<string[]>(["a", "x"]);
        setValue = setState;
        return <Set.Context value={value}>{children}</Set.Context>;
      };
      const { result } = renderHook(
        () => {
          renders++;
          return Set.useMemberAmong(["a", "b"]);
        },
        { wrapper },
      );
      expect(result.current).toEqual("a");
      const baseline = renders;
      act(() => setValue(["a", "y"]));
      expect(renders).toBe(baseline);
      act(() => setValue(["b", "y"]));
      expect(result.current).toEqual("b");
    });
  });
});
