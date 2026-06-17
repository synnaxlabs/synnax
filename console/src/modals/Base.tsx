// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type ReactElement } from "react";

import { type BaseState, useArgs, usePlacer } from "@/modals/Modals";
import { type Renderer } from "@/modals/renderer";

export interface BaseArgs<V> {
  result?: V;
}

export interface BaseProps<Return, Args extends BaseArgs<Return>> {
  value: Args;
  onFinish: (value: Return | null) => void;
}

export interface LayoutOverrides extends Omit<Partial<BaseState>, "type"> {}

export interface Prompt<Return, A extends BaseArgs<Return>> {
  (args: A, layoutOverrides?: LayoutOverrides): Promise<Return | null>;
}

export const createBase = <Return, Args extends BaseArgs<Return>>(
  name: string,
  type: string,
  Component: FC<BaseProps<Return, Args>>,
  defaultLayoutOverrides?: LayoutOverrides,
): [() => Prompt<Return, Args>, Renderer] => {
  const useModal = (): Prompt<Return, Args> => {
    const open = usePlacer();
    return async (args, layoutOverrides) =>
      await open<Return, Args>({
        name,
        type,
        window: { resizable: false, size: { height: 250, width: 700 }, navTop: true },
        ...defaultLayoutOverrides,
        ...layoutOverrides,
        args: { ...args, result: undefined },
      });
  };
  const Modal: Renderer = ({ onClose }): ReactElement => {
    const args = useArgs<Args>();
    return <Component value={args} onFinish={(value) => onClose(value)} />;
  };
  return [useModal, Modal];
};
