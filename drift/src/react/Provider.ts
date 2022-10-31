import { Action, AnyAction, Store } from "@reduxjs/toolkit";
import { ReactElement, createElement, useEffect, useState } from "react";
import { Provider as Base, ProviderProps as BaseProps } from "react-redux";

import { StoreState } from "../state";

/**
 * Overrides the default react-redux Provider to allow for a promise based
 * store.
 */
export interface ProviderProps<
  S extends StoreState,
  A extends Action = AnyAction
> extends Omit<BaseProps<A>, "store"> {
  store: Promise<Store<S, A>>;
}

/**
 * Replaces the default react-redux Provider with a drift compatible one that
 * waits for the store to be ready before rendering. To understand why this is
 * necessary, see the configureStore documentation.
 *
 * @param props - The props to pass to the Provider.
 * @param props.store - A promise that resolves to the store.
 */
export const Provider = <
  S extends StoreState,
  A extends Action<unknown> = AnyAction
>({
  store: promise,
  ...args
}: ProviderProps<S, A>): ReactElement | null => {
  const [store, setStore] = useState<Store<S, A> | null>(null);
  useEffect(() => {
    promise.then((s) => setStore(s)).catch(console.error);
  }, []);
  if (!store) return null;
  return createElement(Base<A>, { ...args, store });
};
