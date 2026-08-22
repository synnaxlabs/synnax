// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux, Ranger } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { useGetSliceState } from "@/session/range/selectors";
import { remove, restore } from "@/session/range/slice";

/**
 * Deletes a range, dropping the copy in the session slice and restoring it, along with
 * the selection it carried, when the server delete fails. A caller's own beforeUpdate
 * runs first and can still cancel the delete.
 */
export const useDelete: Flux.UseUpdate<Ranger.DeleteParams> = (params) => {
  const { beforeUpdate } = params ?? {};
  const getSliceState = useGetSliceState();
  const dispatch = useDispatch();
  return Ranger.useDelete({
    ...params,
    beforeUpdate: useCallback(
      async (args: Flux.BeforeUpdateParams<Ranger.DeleteParams>) => {
        const res = (await beforeUpdate?.(args)) ?? true;
        if (res === false) return false;
        const data = typeof res === "boolean" ? args.data : res;
        const keys = array.toArray(data);
        const { ranges, selected } = getSliceState();
        const removed = ranges
          .map((range, index) => ({ index, range }))
          .filter(({ range }) => keys.includes(range.key));
        if (removed.length === 0) return data;
        dispatch(remove({ keys }));
        const cleared = selected != null && keys.includes(selected);
        args.rollbacks.push(() =>
          dispatch(
            restore({ ranges: removed, selected: cleared ? selected : undefined }),
          ),
        );
        return data;
      },
      [beforeUpdate, getSliceState, dispatch],
    ),
  });
};
