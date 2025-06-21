// Copyright 2025 Synnax Labs, Inc.
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
import { useDispatch } from "react-redux";

import { updateIfExists } from "@/range/slice";

export const useListenForChanges = (): void => {
  const dispatch = useDispatch();
  const handleRangeSet = useCallback(
    (range: ranger.Payload): void => {
      dispatch(updateIfExists(range));
    },
    [dispatch],
  );
  Ranger.useSetSynchronizer(handleRangeSet);
};
