// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MutableRefObject, useCallback, useRef } from "react";

export type PseudoSetState<T> = (value: T | ((prev: T) => T)) => void;

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
