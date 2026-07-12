// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, useMemo } from "react";

import { context } from "@/context";
import { useInitializerRef } from "@/hooks";
import { Store } from "@/store";

/**
 * ContextValue is the registry a {@link Context} shares between portal parts:
 * In parts register their content element under a key, Out parts resolve and
 * subscribe to it.
 */
export interface ContextValue extends Pick<
  Store.UseKeyedListenersReturn<string>,
  "subscribe"
> {
  /** register makes el resolvable under key, replacing any prior entry. */
  register: (key: string, el: HTMLElement) => void;
  /** unregister removes the entry under key. */
  unregister: (key: string) => void;
  /** get resolves the element registered under key, if any. */
  get: (key: string) => HTMLElement | undefined;
}

const [Base, useContext] = context.create<ContextValue>({
  displayName: "Portal.Context",
  providerName: "Portal.Context",
});

export { useContext };

export interface ContextProps extends PropsWithChildren {}

/**
 * Context owns the key to element registry that links In and Out parts. Every
 * In and the Out parts that host its content must share a Context.
 */
export const Context = ({ children }: ContextProps): ReactElement => {
  const { notifyListeners, subscribe } = Store.useKeyedListeners<string>();
  const nodesRef = useInitializerRef(() => new Map<string, HTMLElement>());
  const registry = useMemo<ContextValue>(
    () => ({
      register: (key, el) => {
        nodesRef.current.set(key, el);
        notifyListeners(key);
      },
      unregister: (key) => {
        nodesRef.current.delete(key);
        notifyListeners(key);
      },
      get: (key) => nodesRef.current.get(key),
      subscribe,
    }),
    [notifyListeners, subscribe],
  );
  return <Base value={registry}>{children}</Base>;
};
