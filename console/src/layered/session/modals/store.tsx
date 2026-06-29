// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { type ComponentType, type ReactNode } from "react";

/** The params of a modal that takes no params. */
type Empty = Record<never, never>;

/**
 * The props handed to every modal content component: the modal's typed params spread
 * alongside a single close callback. Calling close with a result resolves the caller's
 * promise (for prompts); calling it with no argument dismisses without a result.
 */
export type ContentProps<Params = Empty, Result = unknown> = Params & {
  close: (result?: Result) => void;
};

/** A component that renders a modal's content of a particular params/result type. */
export type Content<Params = Empty, Result = unknown> = ComponentType<
  ContentProps<Params, Result>
>;

/**
 * A single open modal. Its render and dismiss closures already have the modal's params
 * and result type bound (at push time, where they are known), so the entry itself is
 * untyped — letting the store hold a heterogeneous stack without erasing through
 * assertions. The closures cannot live in serializable state, which is why the stack
 * cannot live in Redux.
 */
export interface Entry {
  /** A unique identity for the open modal. */
  key: string;
  /** Renders the modal's content with its bound params and close callback. */
  render: () => ReactNode;
  /** Dismisses the modal, resolving its caller with null. */
  dismiss: () => void;
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
   * push opens a modal that renders Component with the given params bound, appending it to
   * the top of the stack. resolve settles the caller's promise: it receives the result the
   * content passes to close, or null on dismissal.
   */
  push<Params, Result>(
    Component: Content<Params, Result>,
    params: Params | undefined,
    resolve: (result: Result | null) => void,
  ): void {
    const key = id.create();
    // params is only absent when every field is optional (see optional.Arg in the
    // factory), so an empty object is a valid Params.
    const bound = params ?? ({} as Params);
    const settle = (result: Result | null): void => {
      this.emit(this.stack.filter((e) => e.key !== key));
      resolve(result);
    };
    this.emit([
      ...this.stack,
      {
        key,
        render: () => <Component {...bound} close={(result) => settle(result ?? null)} />,
        dismiss: () => settle(null),
      },
    ]);
  }

  /** closeTop dismisses the topmost open modal, resolving its caller with null. */
  closeTop(): void {
    this.stack.at(-1)?.dismiss();
  }

  /** clear dismisses every open modal, resolving each caller with null. */
  clear(): void {
    this.stack.forEach((e) => e.dismiss());
  }
}

/**
 * The window-local modal stack. Imported directly by non-React consumers (keyboard
 * triggers, the active-tab blur selector) that must read modal state synchronously.
 */
export const modalStore = new ModalStore();
