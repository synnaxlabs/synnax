// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";

export interface BaseArgs<V> {
  result?: V;
}

interface BaseProps<R, A extends BaseArgs<R>> {
  value: A;
  onFinish: (value: R | null) => void;
}

export interface LayoutOverrides
  extends Omit<Partial<Layout.State>, "key" | "type" | "location"> {}

export interface Prompt<R, A extends BaseArgs<R>> {
  (args: A, layoutOverrides?: LayoutOverrides): Promise<R | null>;
}

export const createBase = <R, A extends BaseArgs<R>>(
  name: string,
  type: string,
  Component: FC<BaseProps<R, A>>,
): [() => Prompt<R, A>, Layout.Renderer] => {
  const configureLayout = (
    args: A,
    layoutOverrides?: LayoutOverrides,
  ): Layout.State<A> => ({
    name,
    type,
    key: type,
    windowKey: type,
    location: "modal",
    window: { resizable: false, size: { height: 250, width: 700 }, navTop: true },
    ...layoutOverrides,
    args: { ...args, result: undefined },
  });
  const useModal = (): Prompt<R, A> => {
    const placeLayout = Layout.usePlacer();
    const store = useStore<Layout.StoreState>();
    return async (
      args: A,
      layoutOverrides?: Omit<Partial<Layout.State>, "key" | "type">,
    ) => {
      let unsubscribe: ReturnType<typeof store.subscribe> | null = null;
      return await new Promise((resolve) => {
        const layout = configureLayout(args, layoutOverrides);
        placeLayout(layout);
        const { key } = layout;
        unsubscribe = store.subscribe(() => {
          const state = store.getState();
          const l = Layout.select(state, key);
          if (l == null) resolve(null);
          const args = Layout.selectArgs<A>(state, key);
          if (args?.result == null) resolve(null);
          else resolve(args.result);
          unsubscribe?.();
        });
      });
    };
  };
  const Modal: Layout.Renderer = ({ layoutKey, onClose }) => {
    const args = Layout.useSelectArgs<A>(layoutKey);
    const dispatch = useDispatch();
    const handleResult = (value: R | null) => {
      if (value == null) return onClose();
      dispatch(
        Layout.setArgs<BaseArgs<R>>({
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
