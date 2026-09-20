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

/** A state value and a setter that accepts a value or an updater. */
export type UseReturn<NextState extends state.State> = [
  NextState,
  state.Setter<NextState>,
];
export type Use = <NextState extends state.State>(
  initial: state.Initial<NextState>,
) => UseReturn<NextState>;
/** A state value and a setter that accepts only a value, never an updater. */
export type PureUseReturn<NextState extends state.State> = [
  NextState,
  state.PureSetter<NextState>,
];
export type PureUse<NextState extends state.State> = (
  initial: NextState,
) => PureUseReturn<NextState>;

/** Props for {@link usePassthrough}. */
export interface UsePassthroughProps<NextState extends state.State> {
  initial: state.Initial<NextState>;
  /** Set it, with `onChange`, to let the caller own the state. */
  value?: NextState;
  onChange?: state.Setter<NextState>;
}

/**
 * Lets a component be controlled or uncontrolled through one API: the caller owns the
 * state when it passes both `value` and `onChange`, and the component owns it
 * otherwise. `onChange` fires either way, so `value` decides who owns the state, not
 * who hears about it.
 */
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

/** Props for {@link usePurePassthrough}. */
export interface UsePurePassthroughProps<NextState extends state.State> {
  initialValue: state.Initial<NextState>;
  value?: NextState;
  onChange?: state.PureSetter<NextState>;
}

/** {@link usePassthrough} for a setter that takes only values, never updaters. */
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

/** State backed by local storage under the given key, restored on the next mount. */
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
