// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Primitive, type UnknownRecord } from "@synnaxlabs/x";
import { useId, useState } from "react";

export type State = Primitive | UnknownRecord;
export type SetFunc<S, PS = S> = (prev: PS) => S;

export const isSetter = <S extends State>(arg: SetArg<S>): arg is SetFunc<S> =>
  typeof arg === "function";

export type SetArg<S extends State, PS = S> = S | SetFunc<S, PS>;
export type Set<S extends State> = (value: SetArg<S>) => void;
export type PureSet<S extends State> = (value: S) => void;
export type Initial<S extends State> = S | (() => S);

export const executeSetter = <S extends State>(setter: SetArg<S>, prev: S): S =>
  isSetter(setter) ? setter(prev) : setter;

export const executeInitialSetter = <S extends State>(setter: Initial<S>): S =>
  isInitialSetter(setter) ? setter() : setter;

export const isInitialSetter = <S extends State>(arg: Initial<S>): arg is () => S =>
  typeof arg === "function";

export type UseReturn<S extends State> = [S, Set<S>];
export type Use = <S extends State>(initial: Initial<S>) => UseReturn<S>;
export type PureUseReturn<S extends State> = [S, PureSet<S>];
export type PureUse<S extends State> = (initial: S) => PureUseReturn<S>;

export interface UsePassthroughProps<S extends State> {
  initial: Initial<S>;
  value?: S;
  onChange?: Set<S>;
}

export const usePassthrough = <S extends State>({
  initial,
  value,
  onChange,
}: UsePassthroughProps<S>): UseReturn<S> => {
  const [internal, setInternal] = useState<S>(executeInitialSetter(value ?? initial));
  if (value != null && onChange != null) return [value, onChange];
  return [internal, setInternal];
};

export interface UsePurePassthroughProps<S extends State> {
  initial: Initial<S>;
  value?: S;
  onChange?: PureSet<S>;
  callOnChangeIfValueIsUndefined?: boolean;
}

export const usePurePassthrough = <S extends State>({
  initial,
  value,
  onChange,
}: UsePurePassthroughProps<S>): PureUseReturn<S> => {
  const [internal, setInternal] = useState<S>(executeInitialSetter(value ?? initial));
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
