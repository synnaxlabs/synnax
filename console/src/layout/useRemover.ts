// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status, useMemoCompare } from "@synnaxlabs/pluto";
import { compare, unique } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { select } from "@/layout/selectors";
import { remove } from "@/layout/slice";
import { type State } from "@/layout/types";
import { useConfirm } from "@/modals/Confirm";
import { type RootState } from "@/store";

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
  const store = useStore<RootState>();
  const promptConfirm = useConfirm();
  const handleError = Status.useErrorHandler();
  const memoKeys = useMemoCompare(
    () => baseKeys,
    ([a], [b]) => compare.primitiveArrays(a, b) === compare.EQUAL,
    [baseKeys],
  );

  return useCallback(
    (...keysAlt): void => {
      const keys = unique.unique([...keysAlt, ...memoKeys]);
      const unsavedLayouts = keys
        .map((key) => select(store.getState(), key))
        .filter((layout) => layout != null && layout.unsavedChanges == true) as State[];
      if (unsavedLayouts.length == 0) {
        dispatch(remove({ keys }));
        return;
      }
      handleError(async () => {
        const results: (boolean | null)[] = [];
        for (const layout of unsavedLayouts) {
          const { name, icon } = layout;
          let message = `${name} has unsaved changes. Are you sure you want to close it?`;
          if (name.includes(".")) message = `Are you sure you want to exit?`;
          const result = await promptConfirm(
            {
              message,
              description: "Any unsaved changes will be lost.",
            },
            { icon, name: `${name}.Lose Unsaved Changes` },
          );
          results.push(result);
        }
        dispatch(remove({ keys: keys.filter((_, i) => results[i] === true) }));
      }, "Failed to remove layouts");
    },
    [memoKeys, dispatch, store, promptConfirm],
  );
};
