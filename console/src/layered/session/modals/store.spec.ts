// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { type Entry, ModalStore } from "@/layered/session/modals/store";

const entry = (key: string, resolve: (r: unknown) => void = () => {}): Entry => ({
  key,
  Renderer: () => null,
  params: undefined,
  resolve,
});

describe("ModalStore", () => {
  describe("push", () => {
    it("should append an entry to the stack", () => {
      const store = new ModalStore();
      store.push(entry("a"));
      store.push(entry("b"));
      expect(store.getState().map((e) => e.key)).toEqual(["a", "b"]);
    });

    it("should dedupe a push with an already-open key and resolve it null", () => {
      const store = new ModalStore();
      store.push(entry("a"));
      const resolve = vi.fn();
      store.push(entry("a", resolve));
      expect(store.getState()).toHaveLength(1);
      expect(resolve).toHaveBeenCalledWith(null);
    });
  });

  describe("close", () => {
    it("should remove the entry and resolve with the result", () => {
      const store = new ModalStore();
      const resolve = vi.fn();
      store.push(entry("a", resolve));
      store.close("a", 42);
      expect(store.isAnyOpen()).toBe(false);
      expect(resolve).toHaveBeenCalledWith(42);
    });

    it("should resolve null when closed without a result", () => {
      const store = new ModalStore();
      const resolve = vi.fn();
      store.push(entry("a", resolve));
      store.close("a");
      expect(resolve).toHaveBeenCalledWith(null);
    });

    it("should ignore an unknown key", () => {
      const store = new ModalStore();
      store.push(entry("a"));
      store.close("missing");
      expect(store.getState()).toHaveLength(1);
    });
  });

  describe("closeTop", () => {
    it("should dismiss only the topmost entry with null", () => {
      const store = new ModalStore();
      const resolveA = vi.fn();
      const resolveB = vi.fn();
      store.push(entry("a", resolveA));
      store.push(entry("b", resolveB));
      store.closeTop();
      expect(store.getState().map((e) => e.key)).toEqual(["a"]);
      expect(resolveB).toHaveBeenCalledWith(null);
      expect(resolveA).not.toHaveBeenCalled();
    });
  });

  describe("clear", () => {
    it("should dismiss every entry with null", () => {
      const store = new ModalStore();
      const resolveA = vi.fn();
      const resolveB = vi.fn();
      store.push(entry("a", resolveA));
      store.push(entry("b", resolveB));
      store.clear();
      expect(store.isAnyOpen()).toBe(false);
      expect(resolveA).toHaveBeenCalledWith(null);
      expect(resolveB).toHaveBeenCalledWith(null);
    });
  });

  describe("subscribe", () => {
    it("should notify listeners on push and close", () => {
      const store = new ModalStore();
      const listener = vi.fn();
      const unsubscribe = store.subscribe(listener);
      store.push(entry("a"));
      store.close("a");
      expect(listener).toHaveBeenCalledTimes(2);
      unsubscribe();
      store.push(entry("b"));
      expect(listener).toHaveBeenCalledTimes(2);
    });
  });
});
