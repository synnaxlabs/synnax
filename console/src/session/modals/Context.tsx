// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { id, type record } from "@synnaxlabs/x";
import {
  type ComponentType,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useState,
  useSyncExternalStore,
} from "react";

/**
 * The props handed to every modal content component: the modal's typed params spread
 * alongside a single close callback. Calling close with a result resolves the caller's
 * promise (for prompts); calling it with no argument dismisses without a result.
 */
export type ContentProps<Params = record.Empty, Result = unknown> = Params & {
  close: (result?: Result) => void;
};

/** A component that renders a modal's content of a particular params/result type. */
export type Content<Params = record.Empty, Result = unknown> = ComponentType<
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
 * An in-memory, per-window stack of open modals. Created once by {@link Provider} and
 * read via {@link useStore}. The store is both a React external store (via
 * {@link subscribe}/{@link getState}) and imperatively readable (via {@link isAnyOpen})
 * for event-handler consumers that cannot use hooks.
 */
export class Store {
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
    const settle = (result?: Result): void => {
      this.emit(this.stack.filter((e) => e.key !== key));
      resolve(result ?? null);
    };
    const entry = {
      key,
      render: () => <Component {...bound} close={settle} />,
      dismiss: settle,
    };
    this.emit([...this.stack, entry]);
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

const [Context, useStore] = context.create<Store>({
  displayName: "Modal.StoreContext",
  providerName: "Modal.StoreProvider",
});

export { useStore };

/**
 * Provider creates the window-local modal store and makes it available to descendants via
 * context. Mount once near the root of each window; the store lives for the window's
 * lifetime and is never shared across windows or persisted.
 */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const [store] = useState(() => new Store());
  return <Context value={store}>{children}</Context>;
};

/** useStack subscribes to and returns the current modal stack. */
export const useStack = (): readonly Entry[] => {
  const store = useStore("useStack");
  return useSyncExternalStore(store.subscribe, store.getState);
};
