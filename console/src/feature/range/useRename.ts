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

import { Session } from "@/session";

/**
 * Renames a range on the server and keeps the favorited copy in the session slice in
 * sync, rolling the slice back when the server write fails.
 */
export const useRename = () => {
  const getRangeState = Session.Range.useGetState();
  const dispatch = Session.useDispatch();
  return Ranger.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Ranger.RenameParams>) => {
        const { key, name } = data;
        const rng = getRangeState(key);
        if (rng == null) return data;
        const oldName = rng.name;
        if (!rng.persisted) return false;
        dispatch(Session.Range.rename({ key, name }));
        rollbacks.push(() => dispatch(Session.Range.rename({ key, name: oldName })));
        return data;
      },
      [getRangeState],
    ),
  });
};
