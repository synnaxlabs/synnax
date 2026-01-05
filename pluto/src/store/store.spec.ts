// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useKeyedListeners } from "@/store/store";

describe("useKeyedListeners", () => {
  describe("initialization", () => {
    it("should return notifyListeners and subscribe functions", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      expect(result.current.notifyListeners).toBeInstanceOf(Function);
      expect(result.current.subscribe).toBeInstanceOf(Function);
    });

    it("should maintain stable references across re-renders", () => {
      const { result, rerender } = renderHook(() => useKeyedListeners<string>());
      const initialNotifyListeners = result.current.notifyListeners;
      const initialSubscribe = result.current.subscribe;

      rerender();

      expect(result.current.notifyListeners).toBe(initialNotifyListeners);
      expect(result.current.subscribe).toBe(initialSubscribe);
    });
  });

  describe("subscribe", () => {
    it("should subscribe a listener without a key", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      act(() => {
        const unsubscribe = result.current.subscribe(listener);
        expect(unsubscribe).toBeInstanceOf(Function);
      });

      act(() => {
        result.current.notifyListeners(["key1", "key2"]);
      });

      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should subscribe a listener with a specific key", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      act(() => {
        result.current.subscribe(listener, "key1");
      });

      act(() => {
        result.current.notifyListeners(["key1"]);
      });

      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should return unsubscribe function that removes listener", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      let unsubscribe: () => void;
      act(() => {
        unsubscribe = result.current.subscribe(listener);
      });

      act(() => {
        result.current.notifyListeners(["key1"]);
      });
      expect(listener).toHaveBeenCalledTimes(1);

      act(() => {
        unsubscribe();
      });

      act(() => {
        result.current.notifyListeners(["key1"]);
      });
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should handle multiple listeners", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener1 = vi.fn();
      const listener2 = vi.fn();

      act(() => {
        result.current.subscribe(listener1);
        result.current.subscribe(listener2, "key1");
      });

      act(() => {
        result.current.notifyListeners(["key1"]);
      });

      expect(listener1).toHaveBeenCalledTimes(1);
      expect(listener2).toHaveBeenCalledTimes(1);
    });
  });

  describe("notifyListeners", () => {
    it("should notify listeners without keys for any notification", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const globalListener = vi.fn();

      act(() => {
        result.current.subscribe(globalListener);
      });

      act(() => {
        result.current.notifyListeners(["key1"]);
      });
      expect(globalListener).toHaveBeenCalledTimes(1);

      act(() => {
        result.current.notifyListeners(["key2", "key3"]);
      });
      expect(globalListener).toHaveBeenCalledTimes(2);
    });

    it("should only notify listeners with matching keys", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener1 = vi.fn();
      const listener2 = vi.fn();
      const listener3 = vi.fn();

      act(() => {
        result.current.subscribe(listener1, "key1");
        result.current.subscribe(listener2, "key2");
        result.current.subscribe(listener3, "key3");
      });

      act(() => {
        result.current.notifyListeners(["key1", "key3"]);
      });

      expect(listener1).toHaveBeenCalledTimes(1);
      expect(listener2).toHaveBeenCalledTimes(0);
      expect(listener3).toHaveBeenCalledTimes(1);
    });

    it("should notify listeners when their key is included in the array", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      act(() => {
        result.current.subscribe(listener, "targetKey");
      });

      act(() => {
        result.current.notifyListeners(["otherKey", "targetKey", "anotherKey"]);
      });

      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should not notify listeners when their key is not included", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      act(() => {
        result.current.subscribe(listener, "targetKey");
      });

      act(() => {
        result.current.notifyListeners(["otherKey", "anotherKey"]);
      });

      expect(listener).toHaveBeenCalledTimes(0);
    });

    it("should handle empty notification array", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const keyedListener = vi.fn();
      const globalListener = vi.fn();

      act(() => {
        result.current.subscribe(keyedListener, "key1");
        result.current.subscribe(globalListener);
      });

      act(() => {
        result.current.notifyListeners([]);
      });

      expect(keyedListener).toHaveBeenCalledTimes(0);
      expect(globalListener).toHaveBeenCalledTimes(1);
    });
  });

  describe("key types", () => {
    it("should work with string keys", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      act(() => {
        result.current.subscribe(listener, "stringKey");
      });

      act(() => {
        result.current.notifyListeners(["stringKey"]);
      });

      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should work with number keys", () => {
      const { result } = renderHook(() => useKeyedListeners<number>());
      const listener = vi.fn();

      act(() => {
        result.current.subscribe(listener, 42);
      });

      act(() => {
        result.current.notifyListeners([42, 100]);
      });

      expect(listener).toHaveBeenCalledTimes(1);
    });
  });

  describe("edge cases", () => {
    it("should log a warning if the same listener is subscribed to twice without being unsubscribed", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      act(() => {
        result.current.subscribe(listener, "key1");
        result.current.subscribe(listener, "key2");
      });

      expect(consoleSpy).toHaveBeenCalledWith(
        "[store] attempted to subscribe listener with key key1 to key key2 without being unsubscribed first",
      );
      consoleSpy.mockRestore();
    });

    it("should handle unsubscribing a non-existent listener gracefully", () => {
      const { result } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();

      let unsubscribe: () => void;
      act(() => {
        unsubscribe = result.current.subscribe(listener);
      });

      act(() => {
        unsubscribe();
        unsubscribe();
      });

      act(() => {
        result.current.notifyListeners(["key1"]);
      });

      expect(listener).toHaveBeenCalledTimes(0);
    });
  });

  describe("cleanup", () => {
    it("should clean up listeners when component unmounts", () => {
      const { result, unmount } = renderHook(() => useKeyedListeners<string>());
      const listener = vi.fn();
      act(() => {
        result.current.subscribe(listener);
      });
      unmount();
      expect(listener).toHaveBeenCalledTimes(0);
    });
  });
});
