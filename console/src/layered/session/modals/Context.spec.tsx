// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Modals } from "@/layered/session/modals";

const Noop: Modals.Content = () => null;

/** Pulls the close callback the store bound into the topmost modal's rendered content. */
const closeOf = (store: Modals.Store): Modals.ContentProps["close"] =>
  (store.getState().at(-1)?.render() as ReactElement<Modals.ContentProps>).props.close;

describe("Store", () => {
  describe("push", () => {
    it("should append a modal to the stack", () => {
      const store = new Modals.Store();
      store.push(Noop, undefined, () => {});
      store.push(Noop, undefined, () => {});
      expect(store.getState()).toHaveLength(2);
    });

    it("should resolve with the result the content passes to close", () => {
      const store = new Modals.Store();
      const resolve = vi.fn();
      store.push(Noop, undefined, resolve);
      closeOf(store)(42);
      expect(store.isAnyOpen()).toBe(false);
      expect(resolve).toHaveBeenCalledWith(42);
    });

    it("should resolve null when the content closes without a result", () => {
      const store = new Modals.Store();
      const resolve = vi.fn();
      store.push(Noop, undefined, resolve);
      closeOf(store)();
      expect(resolve).toHaveBeenCalledWith(null);
    });

    it("should generate a unique key for each pushed modal", () => {
      const store = new Modals.Store();
      store.push(Noop, undefined, () => {});
      store.push(Noop, undefined, () => {});
      const [a, b] = store.getState();
      expect(a.key).not.toEqual(b.key);
    });

    it("should bind an empty object when params are omitted", () => {
      const store = new Modals.Store();
      let received: Modals.ContentProps | undefined;
      const Probe: Modals.Content = (props) => {
        received = props;
        return null;
      };
      store.push(Probe, undefined, () => {});
      render(<>{store.getState()[0].render()}</>);
      expect(received).toBeDefined();
      expect(Object.keys(received as object)).toEqual(["close"]);
    });

    it("should bind the provided params onto the content", () => {
      const store = new Modals.Store();
      let received: { label?: string } | undefined;
      const Probe: Modals.Content<{ label: string }> = (props) => {
        received = props;
        return null;
      };
      store.push(Probe, { label: "hello" }, () => {});
      render(<>{store.getState()[0].render()}</>);
      expect(received?.label).toEqual("hello");
    });
  });

  describe("dismiss", () => {
    it("should remove the entry and resolve null", () => {
      const store = new Modals.Store();
      const resolve = vi.fn();
      store.push(Noop, undefined, resolve);
      store.getState()[0].dismiss();
      expect(store.isAnyOpen()).toBe(false);
      expect(resolve).toHaveBeenCalledWith(null);
    });
  });

  describe("closeTop", () => {
    it("should dismiss only the topmost entry with null", () => {
      const store = new Modals.Store();
      const resolveA = vi.fn();
      const resolveB = vi.fn();
      store.push(Noop, undefined, resolveA);
      store.push(Noop, undefined, resolveB);
      store.closeTop();
      expect(store.getState()).toHaveLength(1);
      expect(resolveB).toHaveBeenCalledWith(null);
      expect(resolveA).not.toHaveBeenCalled();
    });

    it("should be a no-op when the stack is empty", () => {
      const store = new Modals.Store();
      expect(() => store.closeTop()).not.toThrow();
      expect(store.isAnyOpen()).toBe(false);
    });
  });

  describe("clear", () => {
    it("should dismiss every entry with null", () => {
      const store = new Modals.Store();
      const resolveA = vi.fn();
      const resolveB = vi.fn();
      store.push(Noop, undefined, resolveA);
      store.push(Noop, undefined, resolveB);
      store.clear();
      expect(store.isAnyOpen()).toBe(false);
      expect(resolveA).toHaveBeenCalledWith(null);
      expect(resolveB).toHaveBeenCalledWith(null);
    });
  });

  describe("subscribe", () => {
    it("should notify listeners on push and dismiss", () => {
      const store = new Modals.Store();
      const listener = vi.fn();
      const unsubscribe = store.subscribe(listener);
      store.push(Noop, undefined, () => {});
      store.getState()[0].dismiss();
      expect(listener).toHaveBeenCalledTimes(2);
      unsubscribe();
      store.push(Noop, undefined, () => {});
      expect(listener).toHaveBeenCalledTimes(2);
    });

    it("should notify every subscribed listener", () => {
      const store = new Modals.Store();
      const a = vi.fn();
      const b = vi.fn();
      store.subscribe(a);
      store.subscribe(b);
      store.push(Noop, undefined, () => {});
      expect(a).toHaveBeenCalledTimes(1);
      expect(b).toHaveBeenCalledTimes(1);
    });
  });
});

const wrapper = ({ children }: PropsWithChildren) => (
  <Modals.Provider>{children}</Modals.Provider>
);

describe("Provider", () => {
  it("should throw when useStore is used outside a Provider", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => Modals.useStore("test"))).toThrow();
    consoleError.mockRestore();
  });

  it("should provide a stable store across re-renders", () => {
    const { result, rerender } = renderHook(() => Modals.useStore("test"), {
      wrapper,
    });
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });
});

describe("useStack", () => {
  it("should reactively reflect pushes and dismissals", () => {
    const { result } = renderHook(
      () => ({ stack: Modals.useStack(), store: Modals.useStore("test") }),
      { wrapper },
    );
    expect(result.current.stack).toHaveLength(0);
    act(() => result.current.store.push(Noop, undefined, () => {}));
    expect(result.current.stack).toHaveLength(1);
    act(() => result.current.store.push(Noop, undefined, () => {}));
    expect(result.current.stack).toHaveLength(2);
    act(() => result.current.store.closeTop());
    expect(result.current.stack).toHaveLength(1);
    act(() => result.current.store.clear());
    expect(result.current.stack).toHaveLength(0);
  });
});
