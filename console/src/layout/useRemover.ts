// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoCompare } from "@synnaxlabs/pluto";
import { compare } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { remove } from "@/layout/slice";

/** A function that removes a layout. */
export interface Remover {
  (...keys: string[]): void;
}

/**
 * useLayoutRemover is a hook that returns a function that allows the caller to remove
 * a layout.
 *
 * @param key - The key of the layout to remove.
 * @returns A layout remover function that allows the caller to remove a layout. If
 * the layout is in a window, the window will also be closed.
 */
export const useRemover = (...baseKeys: string[]): Remover => {
  const dispatch = useDispatch();
  const memoKeys = useMemoCompare(
    () => baseKeys,
    ([a], [b]) => compare.primitiveArrays(a, b) === compare.EQUAL,
    [baseKeys],
  );
  return useCallback(
    (...keys) => dispatch(remove({ keys: [...keys, ...memoKeys] })),
    [memoKeys],
  );
};
