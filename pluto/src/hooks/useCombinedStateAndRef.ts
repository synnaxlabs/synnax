// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef, useState } from "react";

import { type Primitive } from "@synnaxlabs/x";

import { state } from "@/state";

export const useCombinedStateAndRef = <T extends Primitive | object>(
  initialState: state.Initial<T>,
): [T, state.Set<T>, React.MutableRefObject<T>] => {
  const ref = useRef<T | null>(null);
  const [s, setS] = useState<T>(() => {
    const s = state.executeInitialSetter<T>(initialState);
    ref.current = s;
    return s;
  });

  const setStateAndRef: state.Set<T> = useCallback(
    (nextState): void => {
      setS((p) => {
        ref.current = state.executeSetter<T>(nextState, p);
        return ref.current;
      });
    },
    [setS],
  );

  return [s, setStateAndRef, ref as React.MutableRefObject<T>];
};
