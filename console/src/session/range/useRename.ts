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
 * Renames a range. A range the session owns is renamed in the slice alone, since the
 * Core has no copy of it; the Core's own ranges are renamed there, which is where
 * their name lives.
 */
export const useRename = () => {
  const getState = useGetState();
  const dispatch = useDispatch();
  return Ranger.useRename({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<Ranger.RenameParams>) => {
        const { key, name } = data;
        const rng = getState(key);
        if (rng == null || rng.variant === "persisted") return data;
        dispatch(rename({ key, name }));
        return false;
      },
      [getState, dispatch],
    ),
  });
};
