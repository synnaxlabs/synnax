// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { state } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

export type UseReturn<NextState extends state.State> = [
  NextState,
  state.Setter<NextState>,
];
export type Use = <NextState extends state.State>(
  initial: state.Initial<NextState>,
) => UseReturn<NextState>;
export type PureUseReturn<NextState extends state.State> = [
  NextState,
  state.PureSetter<NextState>,
];
export type PureUse<NextState extends state.State> = (
  initial: NextState,
) => PureUseReturn<NextState>;

export interface UsePassthroughProps<NextState extends state.State> {
  initial: state.Initial<NextState>;
  value?: NextState;
  onChange?: state.Setter<NextState>;
}

// usePassthrough always notifies onChange of a change, whether or not the state is
// controlled: value decides who owns the state, not who hears about it. In
// uncontrolled mode the setter's argument is forwarded to onChange as-is, matching
// controlled mode, where onChange is the setter and receives function updaters
// directly.
export const usePassthrough = <NextState extends state.State>({
  initial,
  value,
  onChange,
}: UsePassthroughProps<NextState>): UseReturn<NextState> => {
  const [internal, setInternal] = useState(value ?? initial);
  const setAndNotify = useCallback(
    (arg: state.SetArg<NextState>) => {
      setInternal(arg);
      onChange?.(arg);
    },
    [onChange],
  );
  if (value != null && onChange != null) return [value, onChange];
  return [internal, setAndNotify];
};

export interface UsePurePassthroughProps<NextState extends state.State> {
  initialValue: state.Initial<NextState>;
  value?: NextState;
  onChange?: state.PureSetter<NextState>;
}

// usePurePassthrough has the same notification contract as usePassthrough.
export const usePurePassthrough = <NextState extends state.State>({
  initialValue,
  value,
  onChange,
}: UsePurePassthroughProps<NextState>): PureUseReturn<NextState> => {
  const [internal, setInternal] = useState<NextState>(
    state.executeInitialSetter(value ?? initialValue),
  );
  const setAndNotify = useCallback(
    (next: NextState) => {
      setInternal(next);
      onChange?.(next);
    },
    [onChange],
  );
  if (value != null && onChange != null) return [value, onChange];
  return [internal, setAndNotify];
};

export const usePersisted = <S extends state.State>(
  initial: state.Initial<S>,
  key: string,
): UseReturn<S> => {
  const [internal, setInternal] = useState<S>(() => {
    const stored = localStorage.getItem(key);
    if (stored == null) return state.executeInitialSetter(initial);
    return JSON.parse(stored);
  });
  const set = useCallback(
    (value: state.SetArg<S>) => {
      setInternal(value);
      localStorage.setItem(key, JSON.stringify(value));
    },
    [setInternal, key],
  );
  return [internal, set];
};
