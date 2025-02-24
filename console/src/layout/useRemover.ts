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
import { useDispatch, useStore } from "react-redux";

import { selectRequired } from "@/layout/selectors";
import { remove } from "@/layout/slice";
import { Modals } from "@/modals";
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
  const promptConfirm = Modals.useConfirm();
  const memoKeys = useMemoCompare(
    () => baseKeys,
    ([a], [b]) => compare.primitiveArrays(a, b) === compare.EQUAL,
    [baseKeys],
  );
  return useCallback(
    (...keys) => {
      const allKeys = [...keys, ...memoKeys];
      const confirmations: (() => Promise<boolean | null>)[] = [];
      for (const key of allKeys) {
        const layout = selectRequired(store.getState(), key);
        const { unsavedChanges, name, icon } = layout;
        console.log(unsavedChanges, name, icon);
        if (unsavedChanges == true)
          confirmations.push(
            async () =>
              await promptConfirm(
                {
                  message: `${name} has unsaved changes. Are you sure you want to close it?`,
                  description: "Any unsaved changes will be lost.",
                },
                { icon, name: `${name}.Lose Unsaved Changes` },
              ),
          );
      }
      if (confirmations.length === 0) {
        dispatch(remove({ keys: allKeys }));
        return;
      }
      void (async () => {
        const results: boolean[] = [];
        for (const confirmation of confirmations) {
          const result = await confirmation();
          if (result != null) results.push(result);
        }
        dispatch(remove({ keys: allKeys.filter((_, i) => results[i] === true) }));
      })();
    },
    [memoKeys, dispatch, store, promptConfirm],
  );
};
