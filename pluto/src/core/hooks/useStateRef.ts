// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MutableRefObject, useCallback, useRef } from "react";

/** A function that mimics the behavior of a setState function from a usetState hook. */
export type PsuedoSetStateArg<S> = S | ((prev: S) => S);
export type PseudoSetState<S> = (value: PsuedoSetStateArg<S>) => void;
export type PseudoInitialState<S> = S | (() => S);

/**
 * A hook that returns a ref and a pseudo-setState function to mimic the
 * behavior of useState but with a ref instead of state.
 *
 * @param initialValue - The initial value of the ref.
 * @returns a tuple containing the ref and the pseudo-setState function.
 */
export const useStateRef = <T extends object>(
  initialValue: T
): [MutableRefObject<T>, PseudoSetState<T>] => {
  const ref = useRef<T>(initialValue);
  const setValue: PseudoSetState<T> = useCallback((value) => {
    if (typeof value === "function") ref.current = value(ref.current);
    else ref.current = value;
  }, []);
  return [ref, setValue];
};
