// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Status, useMemoCompare } from "@synnaxlabs/pluto";
import { compare, unique } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Modals } from "@/platform/modals";
import { Session } from "@/session";

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
  const store = Session.useStore();
  const promptConfirm = Modals.useConfirm();
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
        .map((key) => Session.Layout.select(store.getState(), key))
        .filter(
          (layout) => layout != null && layout.unsavedChanges == true,
        ) as Session.Layout.State[];
      if (unsavedLayouts.length == 0) {
        dispatch(Session.Layout.remove({ keys }));
        return;
      }
      handleError(async () => {
        const results: (boolean | null)[] = [];
        for (const layout of unsavedLayouts) {
          const { name, icon } = layout;
          let message = `${name} has unsaved changes. Are you sure you want to close it?`;
          if (name.includes(".")) message = `Are you sure you want to exit?`;
          const result = await promptConfirm({
            message,
            description: "Any unsaved changes will be lost.",
            title: `${name}.Lose Unsaved Changes`,
            icon: Icon.resolve(icon),
          });
          results.push(result);
        }
        dispatch(
          Session.Layout.remove({ keys: keys.filter((_, i) => results[i] === true) }),
        );
      }, "Failed to remove layouts");
    },
    [memoKeys, dispatch, store, promptConfirm],
  );
};
