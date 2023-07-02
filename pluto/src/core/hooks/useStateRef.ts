// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MutableRefObject, useCallback, useRef } from "react";

import { Primitive, UnknownRecord } from "@synnaxlabs/x";

type State = Primitive | UnknownRecord;
export type StateSetter<S, PS = S> = (prev: PS) => S;

export const isStateSetter = <S extends State>(
  arg: PsuedoSetStateArg<S>
): arg is StateSetter<S> => typeof arg === "function";

/** A function that mimics the behavior of a setState function from a usetState hook. */
export type PsuedoSetStateArg<S extends State, PS = S> = S | StateSetter<S, PS>;
export type PseudoSetState<S extends State> = (value: PsuedoSetStateArg<S>) => void;
export type PseudoInitialState<S extends Primitive | object> = S | (() => S);

/**
 * A hook that returns a ref and a pseudo-setState function to mimic the
 * behavior of useState but with a ref instead of state.
 *
 * @param initialValue - The initial value of the ref.
 * @returns a tuple containing the ref and the pseudo-setState function.
 */
export const useStateRef = <T extends Primitive | object>(
  initialValue: T
): [MutableRefObject<T>, PseudoSetState<T>] => {
  const ref = useRef<T>(initialValue);
  const setValue: PseudoSetState<T> = useCallback((value) => {
    ref.current = isStateSetter(value) ? value(ref.current) : value;
  }, []);
  return [ref, setValue];
};
