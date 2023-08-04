// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useRef, useState } from "react";

import { Primitive } from "@synnaxlabs/x";

import {
  InitialState,
  SetState,
  executeInitialSetter,
  executeStateSetter,
} from "@/util/state";

export const useCombinedStateAndRef = <T extends Primitive | object>(
  initialState: InitialState<T>
): [T, SetState<T>, React.MutableRefObject<T>] => {
  const ref = useRef<T | null>(null);
  const [state, setState] = useState<T>(() => {
    const s = executeInitialSetter<T>(initialState);
    ref.current = s;
    return s;
  });

  const setStateAndRef: SetState<T> = useCallback(
    (nextState): void => {
      setState((p) => {
        ref.current = executeStateSetter<T>(nextState, p);
        return ref.current;
      });
    },
    [setState]
  );

  return [state, setStateAndRef, ref as React.MutableRefObject<T>];
};
