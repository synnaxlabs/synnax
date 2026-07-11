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
import { type Node } from "@/portal/Node";

/**
 * ContextValue is the registry a {@link Provider} shares between portal parts:
 * In parts register their node under a key, Out parts resolve and subscribe
 * to it.
 */
export interface ContextValue {
  /** register makes node resolvable under key, replacing any prior entry. */
  register: (key: string, node: Node) => void;
  /** unregister removes the entry under key. */
  unregister: (key: string) => void;
  /** get resolves the node registered under key, if any. */
  get: (key: string) => Node | undefined;
  /**
   * subscribe invokes listener on every registry change until the returned
   * function is called.
   */
  subscribe: (listener: () => void) => () => void;
}

const createRegistry = (): ContextValue => {
  const nodes = new Map<string, Node>();
  const listeners = new Set<() => void>();
  const notify = (): void => listeners.forEach((l) => l());
  return {
    register: (key, node) => {
      nodes.set(key, node);
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

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Portal.Context",
  providerName: "Portal.Provider",
});

export { useContext };

export interface ProviderProps extends PropsWithChildren {}

/**
 * Provider owns the key to node registry that links In and Out parts. Every In
 * and the Out parts that host its content must share a Provider.
 */
export const Provider = ({ children }: ProviderProps): ReactElement => {
  const [registry] = useState(createRegistry);
  return <Context value={registry}>{children}</Context>;
};
