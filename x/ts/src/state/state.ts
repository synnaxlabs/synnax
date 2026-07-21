// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive } from "@/primitive";
import { type record } from "@/record";

/** Shape of values storable as state. */
export type State = Exclude<primitive.Value, undefined> | record.Unknown | void;

/** A function that computes the next state from the previous state. */
export type SetFunc<S, PS = S> = (prev: PS) => S;

/** A next state, or a function that computes it from the previous state. */
export type SetArg<NextState extends State, PrevState = NextState> =
  NextState | SetFunc<NextState, PrevState>;

/** Accepts a {@link SetArg} and applies it to the state it manages. */
export type Setter<NextState extends State, PrevState = NextState> = (
  value: SetArg<NextState, PrevState>,
) => void;

/** A {@link Setter} that accepts only literal next states, not updaters. */
export type PureSetter<NextState extends State> = (value: NextState) => void;

/** An initial state, or a function that lazily computes it. */
export type Initial<InitialState extends State> = InitialState | (() => InitialState);

export const isSetter = <S extends State, PS = S>(
  arg: SetArg<S, PS>,
): arg is SetFunc<S, PS> => typeof arg === "function";

export const executeSetter = <
  NextState extends State,
  PrevState extends State = NextState,
>(
  setter: SetArg<NextState, PrevState>,
  prev: PrevState,
): NextState => (isSetter(setter) ? setter(prev) : setter);

export const isInitialSetter = <InitialState extends State>(
  arg: Initial<InitialState>,
): arg is () => InitialState => typeof arg === "function";

export const executeInitialSetter = <InitialState extends State>(
  setter: Initial<InitialState>,
): InitialState => (isInitialSetter(setter) ? setter() : setter);

/** Wraps an updater so it passes undefined through instead of applying f. */
export const skipUndefined =
  <NextState extends State, PrevState extends State = NextState>(
    f: SetFunc<NextState, PrevState>,
  ): SetFunc<NextState | undefined, PrevState | undefined> =>
  (v) =>
    v == null ? undefined : f(v);

/** Wraps an updater so it passes null through instead of applying f. */
export const skipNull =
  <NextState extends State, PrevState extends State = NextState>(
    f: SetFunc<NextState, PrevState>,
  ): SetFunc<NextState | null, PrevState | null> =>
  (v) =>
    v == null ? null : f(v);
