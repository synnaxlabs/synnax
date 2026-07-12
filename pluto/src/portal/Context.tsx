// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, useState } from "react";

import { context } from "@/context";

/**
 * ContextValue is the registry a {@link Context} shares between portal parts:
 * In parts register their content element under a key, Out parts resolve and
 * subscribe to it.
 */
export interface ContextValue {
  /** register makes el resolvable under key, replacing any prior entry. */
  register: (key: string, el: HTMLElement) => void;
  /** unregister removes the entry under key. */
  unregister: (key: string) => void;
  /** get resolves the element registered under key, if any. */
  get: (key: string) => HTMLElement | undefined;
  /**
   * subscribe invokes listener on every registry change until the returned
   * function is called.
   */
  subscribe: (listener: () => void) => () => void;
}

const createRegistry = (): ContextValue => {
  const nodes = new Map<string, HTMLElement>();
  const listeners = new Set<() => void>();
  const notify = (): void => listeners.forEach((l) => l());
  return {
    register: (key, el) => {
      nodes.set(key, el);
      notify();
    },
    unregister: (key) => {
      nodes.delete(key);
      notify();
    },
    get: (key) => nodes.get(key),
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
};

const [Base, useContext] = context.create<ContextValue>({
  displayName: "Portal.Context",
  providerName: "Portal.Context",
});

export { useContext };

export interface ContextProps extends PropsWithChildren {}

/**
 * Context owns the key to node registry that links In and Out parts. Every In
 * and the Out parts that host its content must share a Context.
 */
export const Context = ({ children }: ContextProps): ReactElement => {
  const [registry] = useState(createRegistry);
  return <Base value={registry}>{children}</Base>;
};
