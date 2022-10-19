import { Provider as Base, ProviderProps as BaseProps } from 'react-redux';
import { Action, AnyAction, Store } from '@reduxjs/toolkit';
import { createElement, useState, useEffect } from 'react';

export interface ProviderProps<A extends Action = AnyAction>
  extends Omit<BaseProps<A>, 'store'> {
  store: Promise<Store>;
}

export default function Provider<A extends Action<any> = AnyAction>(
  args: ProviderProps<A>
): React.ReactElement {
  const [store, setStore] = useState<Store<any, A> | undefined>(undefined);
  useEffect(() => {
    args.store.then((store) => setStore(store as Store<any, A>));
  }, []);
  if (store) {
    return createElement(Base<A>, { ...args, store });
  }
  return createElement('div');
}
