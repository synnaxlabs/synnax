// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux, Ranger } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { useGetState } from "@/session/range/selectors";
import { rename } from "@/session/range/slice";

/**
 * Renames a range, keeping the copy in the session slice in sync and rolling the slice
 * back when the server write fails. A local range is renamed in the slice alone, as the
 * server has no copy of it.
 */
export const useRename = () => {
  const getState = useGetState();
  const dispatch = useDispatch();
  return Ranger.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Ranger.RenameParams>) => {
        const { key, name } = data;
        const rng = getState(key);
        if (rng == null) return data;
        const oldName = rng.name;
        dispatch(rename({ key, name }));
        if (!rng.persisted) return false;
        rollbacks.push(() => dispatch(rename({ key, name: oldName })));
        return data;
      },
      [getState, dispatch],
    ),
  });
};
