// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type EnhancedStore, type UnknownAction } from "@reduxjs/toolkit";
import { createElement, type ReactElement, useState } from "react";
import { Provider, type ProviderProps as BaseProps } from "react-redux";

import { type Enhancers } from "@/configureStore";
import { type StoreState } from "@/state";

/**
 * Overrides the default react-redux Provider to allow for a promise based
 * store.
 */
export interface ProviderProps<
  S extends StoreState,
  A extends Action = UnknownAction,
  E extends Enhancers = Enhancers,
> extends Omit<BaseProps<A, S>, "store"> {
  store: Promise<EnhancedStore<S, A, E>> | EnhancedStore<S, A, E>;
  emptyContent?: ReactElement | null;
}

/**
 * Replaces the default react-redux Provider with a drift compatible one that
 * waits for the store to be ready before rendering. To understand why this is
 * necessary, see the configureStore documentation.
 *
 * @param props - The props to pass to the Provider.
 * @param props.store - A promise that resolves to the store.
 */
export const DriftProvider = <
  S extends StoreState,
  A extends Action<string> = UnknownAction,
  E extends Enhancers = Enhancers,
>({
  store: storeOrPromise,
  emptyContent = null,
  children,
}: ProviderProps<S, A, E>): ReactElement | null => {
  const [store, setStore] = useState<EnhancedStore<S, A, E> | null>(null);
  const [error, setError] = useState<Error | null>(null);
  if (error != null) {
    setError(null);
    throw error;
  }
  if (store == null) {
    // if the store isn't a promise, then it's already ready
    if (!(storeOrPromise instanceof Promise)) setStore(storeOrPromise);
    else storeOrPromise.then(setStore).catch(setError);
    return emptyContent;
  }
  // @ts-expect-error - store is guaranteed to be non-null here
  return createElement(Provider<A, S>, { store }, children);
};
