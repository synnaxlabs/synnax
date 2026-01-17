// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive, type record } from "@synnaxlabs/x";
import { useState } from "react";

export type State = Exclude<primitive.Value, undefined> | record.Unknown | void;
export type SetFunc<S, PS = S> = (prev: PS) => S;

export const isSetter = <S extends State, PS = S>(
  arg: SetArg<S, PS>,
): arg is SetFunc<S, PS> => typeof arg === "function";

export type SetArg<NextState extends State, PrevState = NextState> =
  | NextState
  | SetFunc<NextState, PrevState>;
export type Setter<NextState extends State, PrevState = NextState> = (
  value: SetArg<NextState, PrevState>,
) => void;
export type PureSetter<NextState extends State> = (value: NextState) => void;
export type Initial<InitialState extends State> = InitialState | (() => InitialState);

export const executeSetter = <
  NextState extends State,
  PrevState extends State = NextState,
>(
  setter: SetArg<NextState, PrevState>,
  prev: PrevState,
): NextState => (isSetter(setter) ? setter(prev) : setter);

export const skipUndefined =
  <NextState extends State, PrevState extends State = NextState>(
    f: SetFunc<NextState, PrevState>,
  ): SetFunc<NextState | undefined, PrevState | undefined> =>
  (v) =>
    v == null ? undefined : f(v);

export const skipNull =
  <NextState extends State, PrevState extends State = NextState>(
    f: SetFunc<NextState, PrevState>,
  ): SetFunc<NextState | null, PrevState | null> =>
  (v) =>
    v == null ? null : f(v);

export const executeInitialSetter = <InitialState extends State>(
  setter: Initial<InitialState>,
): InitialState => (isInitialSetter(setter) ? setter() : setter);

export const isInitialSetter = <InitialState extends State>(
  arg: Initial<InitialState>,
): arg is () => InitialState => typeof arg === "function";

export type UseReturn<NextState extends State> = [NextState, Setter<NextState>];
export type Use = <NextState extends State>(
  initial: Initial<NextState>,
) => UseReturn<NextState>;
export type PureUseReturn<NextState extends State> = [NextState, PureSetter<NextState>];
export type PureUse<NextState extends State> = (
  initial: NextState,
) => PureUseReturn<NextState>;

export interface UsePassthroughProps<NextState extends State> {
  initial: Initial<NextState>;
  value?: NextState;
  onChange?: Setter<NextState>;
}

export const usePassthrough = <NextState extends State>({
  initial,
  value,
  onChange,
}: UsePassthroughProps<NextState>): UseReturn<NextState> => {
  const [internal, setInternal] = useState(value ?? initial);
  if (value != null && onChange != null) return [value, onChange];
  return [internal, setInternal];
};

export interface UsePurePassthroughProps<NextState extends State> {
  initial: Initial<NextState>;
  value?: NextState;
  onChange?: PureSetter<NextState>;
  callOnChangeIfValueIsUndefined?: boolean;
}

export const usePurePassthrough = <NextState extends State>({
  initial,
  value,
  onChange,
}: UsePurePassthroughProps<NextState>): PureUseReturn<NextState> => {
  const [internal, setInternal] = useState<NextState>(
    executeInitialSetter(value ?? initial),
  );
  if (value != null && onChange != null) return [value, onChange];
  return [internal, setInternal];
};

export const usePersisted = <S extends State>(
  initial: Initial<S>,
  key: string,
): UseReturn<S> => {
  const [internal, setInternal] = useState<S>(() => {
    const stored = localStorage.getItem(key);
    if (stored == null) return executeInitialSetter(initial);
    return JSON.parse(stored);
  });
  return [
    internal,
    (value) => {
      setInternal(value);
      localStorage.setItem(key, JSON.stringify(value));
    },
  ];
};
