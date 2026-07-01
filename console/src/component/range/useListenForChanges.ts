// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Ranger } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Session } from "@/session";
import { remove, updateRemote } from "@/session/range/slice";

export const useListenForChanges = (): void => {
  const dispatch = Session.useDispatch();
  const handleRangeSet = useCallback(
    ({ timeRange, ...rest }: ranger.Payload): void => {
      dispatch(
        updateRemote({
          ...rest,
          parent: undefined,
          timeRange: timeRange.numeric,
        }),
      );
    },
    [dispatch],
  );
  Ranger.useSetSynchronizer(handleRangeSet);
  const handleRangeDelete = useCallback(
    (key: ranger.Key) => {
      dispatch(remove({ keys: [key] }));
    },
    [dispatch],
  );
  Ranger.useDeleteSynchronizer(handleRangeDelete);
};
