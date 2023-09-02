// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MutableRefObject, useCallback, useRef } from "react";

import { type Primitive } from "@synnaxlabs/x";

import { state } from "@/state";

/**
 * A hook that returns a ref and a pseudo-setState function to mimic the
 * behavior of useState but with a ref instead of state.
 *
 * @param initialValue - The initial value of the ref.
 * @returns a tuple containing the ref and the pseudo-setState function.
 */
export const useStateRef = <T extends Primitive | object>(
  initialValue: T,
): [MutableRefObject<T>, state.Set<T>] => {
  const ref = useRef<T>(initialValue);
  const setValue: state.Set<T> = useCallback((setter) => {
    ref.current = state.executeSetter(setter, ref.current);
  }, []);
  return [ref, setValue];
};
