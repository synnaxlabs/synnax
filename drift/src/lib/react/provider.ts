import { Action, AnyAction, Store } from '@reduxjs/toolkit';
import { createElement, ReactElement, useEffect, useState } from 'react';
import { Provider as Base, ProviderProps as BaseProps } from 'react-redux';

export interface ProviderProps<A extends Action = AnyAction>
  extends Omit<BaseProps<A>, 'store'> {
  store: Promise<Store>;
}

export const Provider = <S, A extends Action<unknown> = AnyAction>({
  store: storePromise,
  ...args
}: ProviderProps<A>): ReactElement => {
  const [store, setStore] = useState<Store<S, A> | undefined>(undefined);
  useEffect(() => {
    storePromise.then((store) => setStore(store as Store<S, A>));
  }, []);
  if (store) {
    return createElement(Base<A>, { ...args, store });
  }
  return createElement('div');
};
