// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, createElement, useEffect, useState } from "react";

import type { Action, AnyAction, Store } from "@reduxjs/toolkit";
import { Provider as Base } from "react-redux";
import type {ProviderProps as BaseProps } from 'react-redux'

import { StoreState } from "@/state";


/**
 * Overrides the default react-redux Provider to allow for a promise based
 * store.
 */
export interface ProviderProps<S extends StoreState, A extends Action = AnyAction>
	extends Omit<BaseProps<A>, "store"> {
	store: Promise<Store<S, A>>;
  emptyContent?: JSX.Element;
}

/**
 * Replaces the default react-redux Provider with a drift compatible one that
 * waits for the store to be ready before rendering. To understand why this is
 * necessary, see the configureStore documentation.
 *
 * @param props - The props to pass to the Provider.
 * @param props.store - A promise that resolves to the store.
 */
export const Provider = <S extends StoreState, A extends Action<unknown> = AnyAction>({
	store: promise,
  emptyContent,
	...props
}: ProviderProps<S, A>): ReactElement | null => {
	const [store, setStore] = useState<Store<S, A> | null>(null);
	useEffect(() => {
		promise.then((s) => setStore(s)).catch(console.error);
	}, []);
	if (store == null) return emptyContent ?? null;
	return createElement(Base<A>, { ...props, store });
};
