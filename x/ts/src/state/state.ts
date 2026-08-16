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
export type SetArg<S extends State, PS = S> = S | SetFunc<S, PS>;

/** Accepts a {@link SetArg} and applies it to the state it manages. */
export type Setter<S extends State, PS = S> = (value: SetArg<S, PS>) => void;

/** A {@link Setter} that accepts only literal next states, not updaters. */
export type PureSetter<S extends State> = (value: S) => void;

/** An initial state, or a function that lazily computes it. */
export type Initial<S extends State> = S | (() => S);

export const isSetter = <S extends State, PS = S>(
  arg: SetArg<S, PS>,
): arg is SetFunc<S, PS> => typeof arg === "function";

export const executeSetter = <S extends State, PS extends State = S>(
  setter: SetArg<S, PS>,
  prev: PS,
): S => (isSetter(setter) ? setter(prev) : setter);

export const isInitialSetter = <S extends State>(arg: Initial<S>): arg is () => S =>
  typeof arg === "function";

export const executeInitialSetter = <S extends State>(setter: Initial<S>): S =>
  isInitialSetter(setter) ? setter() : setter;

/** Wraps an updater so it passes undefined through instead of applying f. */
export const skipUndefined =
  <S extends State, PS extends State = S>(
    f: SetFunc<S, PS>,
  ): SetFunc<S | undefined, PS | undefined> =>
  (v) =>
    v == null ? undefined : f(v);

/** Wraps an updater so it passes null through instead of applying f. */
export const skipNull =
  <S extends State, PS extends State = S>(
    f: SetFunc<S, PS>,
  ): SetFunc<S | null, PS | null> =>
  (v) =>
    v == null ? null : f(v);
