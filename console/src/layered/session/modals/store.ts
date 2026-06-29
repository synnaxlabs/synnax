// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentType } from "react";

/**
 * The props handed to every modal renderer. A renderer receives its typed params and a
 * single close callback; calling close with a result resolves the caller's promise (for
 * prompts) and dismisses the modal. Calling close with no argument dismisses without a
 * result.
 */
export type ContentProps<Params = unknown, Result = unknown> = Params & {
  close: (result?: Result) => void;
};

/** A component that renders a modal's body (and its own header) of a particular type. */
export type Content<Params = unknown, Result = unknown> = ComponentType<
  ContentProps<Params, Result>
>;

/**
 * A single open modal. Entries are pushed by {@link ModalStore.push} and live until the
 * modal resolves via {@link ModalStore.close}. Both the render component and the resolve
 * closure are held directly on the entry rather than round-tripped through serializable
 * state, which is why the stack cannot live in Redux. All presentation (title, icon) is
 * owned by {@link Content}; the entry carries only the dynamic, per-open core.
 */
export interface Entry<Params = unknown, Result = unknown> {
  /** A unique identity for the open modal, used to address it for close. */
  key: string;
  /** The component that renders this modal. */
  Renderer: Content<Params, Result>;
  /** The typed params handed to the renderer. */
  params: Params;
  /**
   * Resolves the caller's promise. Called exactly once with the modal's result, or with
   * null when the modal is dismissed without a result. A no-op for fire-and-forget
   * modals opened via the open hook.
   */
  resolve: (result: Result | null) => void;
}

type Listener = () => void;

/**
 * An in-memory, per-window stack of open modals. Because Drift windows are isolated JS
 * contexts, a module-level instance is naturally window-local and is never persisted or
 * synced. The store is both a React external store (via {@link subscribe}/{@link getState})
 * and imperatively readable (via {@link isAnyOpen}) for event-handler consumers that
 * cannot use hooks.
 */
export class ModalStore {
  private stack: readonly Entry[] = [];
  private readonly listeners = new Set<Listener>();

  getState = (): readonly Entry[] => this.stack;

  subscribe = (listener: Listener): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  /** isAnyOpen reports whether at least one modal is currently open in this window. */
  isAnyOpen = (): boolean => this.stack.length > 0;

  private emit(next: readonly Entry[]): void {
    this.stack = next;
    this.listeners.forEach((l) => l());
  }

  /**
   * push opens a modal by appending it to the top of the stack. If a modal with the same
   * key is already open, the push is ignored and the new entry is resolved with null so
   * its caller does not hang.
   */
  push<Args, Result>(typed: Entry<Args, Result>): void {
    const entry = typed as unknown as Entry;
    if (this.stack.some((e) => e.key === entry.key)) {
      entry.resolve(null);
      return;
    }
    this.emit([...this.stack, entry]);
  }

  /**
   * close removes the modal with the given key and resolves its caller's promise. A
   * missing or undefined result resolves the caller with null (a dismissal).
   */
  close(key: string, result?: unknown): void {
    const entry = this.stack.find((e) => e.key === key);
    if (entry == null) return;
    this.emit(this.stack.filter((e) => e.key !== key));
    entry.resolve(result === undefined ? null : result);
  }

  /** closeTop dismisses the topmost open modal, resolving its caller with null. */
  closeTop(): void {
    const top = this.stack.at(-1);
    if (top != null) this.close(top.key);
  }

  /** clear dismisses every open modal, resolving each caller with null. */
  clear(): void {
    const entries = this.stack;
    this.emit([]);
    entries.forEach((e) => e.resolve(null));
  }
}

/**
 * The window-local modal stack. Imported directly by non-React consumers (keyboard
 * triggers, the active-tab blur selector) that must read modal state synchronously.
 */
export const modalStore = new ModalStore();
