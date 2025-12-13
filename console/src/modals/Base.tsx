// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { type FC } from "react";
import { useDispatch, useStore } from "react-redux";

import { type Layout } from "@/layout";
import { select, selectArgs, useSelectArgs } from "@/layout/selectors";
import { setArgs } from "@/layout/slice";
import { usePlacer } from "@/layout/usePlacer";

export interface BaseArgs<V> {
  result?: V;
}

export interface BaseProps<R, A extends BaseArgs<R>> {
  value: A;
  onFinish: (value: R | null) => void;
}

export interface LayoutOverrides
  extends Omit<Partial<Layout.BaseState>, "type" | "location"> {}

export interface Prompt<R, A extends BaseArgs<R>> {
  (args: A, layoutOverrides?: LayoutOverrides): Promise<R | null>;
}

export const createBase = <R, A extends BaseArgs<R>>(
  name: string,
  type: string,
  Component: FC<BaseProps<R, A>>,
  defaultLayoutOverrides?: LayoutOverrides,
): [() => Prompt<R, A>, Layout.Renderer] => {
  const configureLayout = (
    key: string,
    args: A,
    layoutOverrides?: LayoutOverrides,
  ): Layout.BaseState<A> & Pick<Layout.State<A>, "key"> => ({
    name,
    type,
    location: "modal",
    window: { resizable: false, size: { height: 250, width: 700 }, navTop: true },
    ...defaultLayoutOverrides,
    ...layoutOverrides,
    key,
    args: { ...args, result: undefined },
  });
  const useModal = (): Prompt<R, A> => {
    const placeLayout = usePlacer();
    const store = useStore<Layout.StoreState>();
    return async (args: A, layoutOverrides?: LayoutOverrides) => {
      let unsubscribe: ReturnType<typeof store.subscribe> | null = null;
      const key = layoutOverrides?.key ?? defaultLayoutOverrides?.key ?? id.create();
      return await new Promise((resolve) => {
        const layout = configureLayout(key, args, layoutOverrides);
        placeLayout(layout);
        unsubscribe = store.subscribe(() => {
          const state = store.getState();
          const l = select(state, key);
          if (l == null) resolve(null);
          const args = selectArgs<A>(state, key);
          if (args?.result == null) return;
          resolve(args.result);
          unsubscribe?.();
        });
      });
    };
  };
  const Modal: Layout.Renderer = ({ layoutKey, onClose }) => {
    const args = useSelectArgs<A>(layoutKey);
    const dispatch = useDispatch();
    const handleResult = (value: R | null) => {
      if (value == null) return onClose();
      dispatch(
        setArgs<BaseArgs<R>>({
          key: layoutKey,
          args: { ...args, result: value },
        }),
      );
      onClose();
    };
    return <Component value={args} onFinish={handleResult} />;
  };
  return [useModal, Modal];
};
