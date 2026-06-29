// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import {
  type Content,
  type ContentProps,
  ModalStore,
} from "@/layered/session/modals/store";

const Noop: Content = () => null;

/** Pulls the close callback the store bound into the topmost modal's rendered content. */
const closeOf = (store: ModalStore): ContentProps["close"] =>
  (store.getState().at(-1)?.render() as ReactElement<ContentProps>).props.close;

describe("ModalStore", () => {
  describe("push", () => {
    it("should append a modal to the stack", () => {
      const store = new ModalStore();
      store.push(Noop, undefined, () => {});
      store.push(Noop, undefined, () => {});
      expect(store.getState()).toHaveLength(2);
    });

    it("should resolve with the result the content passes to close", () => {
      const store = new ModalStore();
      const resolve = vi.fn();
      store.push(Noop, undefined, resolve);
      closeOf(store)(42);
      expect(store.isAnyOpen()).toBe(false);
      expect(resolve).toHaveBeenCalledWith(42);
    });

    it("should resolve null when the content closes without a result", () => {
      const store = new ModalStore();
      const resolve = vi.fn();
      store.push(Noop, undefined, resolve);
      closeOf(store)();
      expect(resolve).toHaveBeenCalledWith(null);
    });
  });

  describe("dismiss", () => {
    it("should remove the entry and resolve null", () => {
      const store = new ModalStore();
      const resolve = vi.fn();
      store.push(Noop, undefined, resolve);
      store.getState()[0].dismiss();
      expect(store.isAnyOpen()).toBe(false);
      expect(resolve).toHaveBeenCalledWith(null);
    });
  });

  describe("closeTop", () => {
    it("should dismiss only the topmost entry with null", () => {
      const store = new ModalStore();
      const resolveA = vi.fn();
      const resolveB = vi.fn();
      store.push(Noop, undefined, resolveA);
      store.push(Noop, undefined, resolveB);
      store.closeTop();
      expect(store.getState()).toHaveLength(1);
      expect(resolveB).toHaveBeenCalledWith(null);
      expect(resolveA).not.toHaveBeenCalled();
    });
  });

  describe("clear", () => {
    it("should dismiss every entry with null", () => {
      const store = new ModalStore();
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
      const store = new ModalStore();
      const listener = vi.fn();
      const unsubscribe = store.subscribe(listener);
      store.push(Noop, undefined, () => {});
      store.getState()[0].dismiss();
      expect(listener).toHaveBeenCalledTimes(2);
      unsubscribe();
      store.push(Noop, undefined, () => {});
      expect(listener).toHaveBeenCalledTimes(2);
    });
  });
});
