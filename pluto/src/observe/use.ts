// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type observe, type Primitive } from "@synnaxlabs/x";
import { useState as reactUseState } from "react";

import { useAsyncEffect } from "@/hooks";
import { useMemoPrimitiveArray } from "@/memo";

export interface UseListenerProps<D> {
  key: Primitive[];
  open?: () => Promise<observe.ObservableAsyncCloseable<D> | undefined>;
  onChange: observe.Handler<D>;
}

export const useListener = <D>({ key, open, onChange }: UseListenerProps<D>) => {
  const memoKey = useMemoPrimitiveArray(key);
  useAsyncEffect(async () => {
    if (open == null) return;
    const obs = await open();
    if (obs == null) return;
    obs.onChange(onChange);
    return async () => await obs.close();
  }, [open == null, memoKey]);
};

export interface UseStateProps<D> extends UseListenerProps<D> {
  fetchInitialValue: () => Promise<D>;
}

interface UseState {}

export const useState = (<D>({ fetchInitialValue, ...props }: UseStateProps<D>) => {
  const [v, setV] = reactUseState<D | undefined>(undefined);
  useAsyncEffect(async () => {
    setV(await fetchInitialValue());
  }, [fetchInitialValue]);
  useListener({ ...props, onChange: setV });
  return v;
}) as UseState;
