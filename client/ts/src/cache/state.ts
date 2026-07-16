// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive, type record } from "@synnaxlabs/x";

/** Shape of values storable in a cache store. */
export type State = Exclude<primitive.Value, undefined> | record.Unknown | void;

/** A function that computes the next state from the previous state. */
export type SetFunc<S, PS = S> = (prev: PS) => S;

/** A next state, or a function that computes it from the previous state. */
export type SetArg<NextState extends State, PrevState = NextState> =
  NextState | SetFunc<NextState, PrevState>;

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
